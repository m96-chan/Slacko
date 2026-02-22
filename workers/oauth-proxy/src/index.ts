/**
 * Slacko OAuth Proxy — Cloudflare Worker
 *
 * Handles the full OAuth callback flow server-side:
 *   GET /authorize  — redirects to Slack with worker as callback
 *   GET /callback   — receives code from Slack, exchanges token, POSTs result to localhost
 *
 * Secrets (set via `wrangler secret put`):
 *   SLACK_CLIENT_ID
 *   SLACK_CLIENT_SECRET
 *   SLACK_APP_TOKEN
 */

interface Env {
	SLACK_CLIENT_ID: string;
	SLACK_CLIENT_SECRET: string;
	SLACK_APP_TOKEN: string;
}

const USER_SCOPES = [
	"channels:history", "channels:read", "channels:write", "chat:write",
	"emoji:read", "files:read", "files:write",
	"groups:history", "groups:read", "groups:write",
	"im:history", "im:read", "im:write",
	"mpim:history", "mpim:read", "mpim:write",
	"pins:read", "pins:write", "reactions:read", "reactions:write",
	"search:read", "stars:read", "stars:write",
	"team:read", "users:read", "users:read.email", "users.profile:read", "users.profile:write",
	"reminders:read", "reminders:write",
].join(",");

export default {
	async fetch(request: Request, env: Env): Promise<Response> {
		const url = new URL(request.url);

		switch (url.pathname) {
			case "/authorize":
				return handleAuthorize(url, env);
			case "/callback":
				return handleCallback(url, env);
			default:
				return new Response("Not Found", { status: 404 });
		}
	},
};

/**
 * GET /authorize?port=PORT&state=CSRF_STATE
 *
 * Encodes port + CSRF state into Slack's state param, then redirects to Slack.
 */
function handleAuthorize(url: URL, env: Env): Response {
	const port = url.searchParams.get("port");
	const csrfState = url.searchParams.get("state");

	if (!port || !csrfState) {
		return new Response("Missing required params: port, state", { status: 400 });
	}

	// Validate port is a number.
	if (!/^\d+$/.test(port)) {
		return new Response("Invalid port", { status: 400 });
	}

	// Encode port and CSRF state together.
	const combinedState = `${port}:${csrfState}`;
	const callbackURL = new URL("/callback", url.origin).toString();

	const params = new URLSearchParams({
		client_id: env.SLACK_CLIENT_ID,
		user_scope: USER_SCOPES,
		redirect_uri: callbackURL,
		state: combinedState,
	});

	return Response.redirect(
		`https://slack.com/oauth/v2/authorize?${params.toString()}`,
		302,
	);
}

/**
 * GET /callback?code=CODE&state=PORT:CSRF_STATE
 *
 * Receives Slack's OAuth callback, exchanges the code for a token,
 * then returns an HTML page that auto-POSTs the result to localhost.
 */
async function handleCallback(url: URL, env: Env): Promise<Response> {
	const error = url.searchParams.get("error");
	if (error) {
		return errorPage(`Slack denied authorization: ${error}`);
	}

	const code = url.searchParams.get("code");
	const state = url.searchParams.get("state");
	if (!code || !state) {
		return errorPage("Missing code or state in callback");
	}

	// Parse combined state: "PORT:CSRF_STATE"
	const colonIdx = state.indexOf(":");
	if (colonIdx === -1) {
		return errorPage("Invalid state format");
	}
	const port = state.substring(0, colonIdx);
	const csrfState = state.substring(colonIdx + 1);

	if (!/^\d+$/.test(port)) {
		return errorPage("Invalid port in state");
	}

	// Exchange code for token.
	const callbackURL = new URL("/callback", url.origin).toString();
	const params = new URLSearchParams({
		client_id: env.SLACK_CLIENT_ID,
		client_secret: env.SLACK_CLIENT_SECRET,
		code,
		redirect_uri: callbackURL,
	});

	const slackResp = await fetch("https://slack.com/api/oauth.v2.access", {
		method: "POST",
		headers: { "Content-Type": "application/x-www-form-urlencoded" },
		body: params.toString(),
	});

	const data = await slackResp.json() as Record<string, unknown>;

	if (!data.ok) {
		const errMsg = (data.error as string) || "token exchange failed";
		return errorPage(`Token exchange failed: ${errMsg}`);
	}

	const authedUser = data.authed_user as Record<string, string> | undefined;
	const team = data.team as Record<string, string> | undefined;
	const userToken = authedUser?.access_token || "";
	const userId = authedUser?.id || "";
	const teamId = team?.id || "";
	const teamName = team?.name || "";

	if (!userToken) {
		return errorPage("No user token in Slack response");
	}

	// Return an HTML page that auto-submits a form POST to the CLI's local server.
	const localhostURL = `http://localhost:${port}/done`;
	const html = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Slacko — Authorizing...</title></head>
<body>
<p>Completing authorization...</p>
<form id="f" method="POST" action="${escapeHtml(localhostURL)}">
<input type="hidden" name="token" value="${escapeHtml(userToken)}">
<input type="hidden" name="user_id" value="${escapeHtml(userId)}">
<input type="hidden" name="team_id" value="${escapeHtml(teamId)}">
<input type="hidden" name="team_name" value="${escapeHtml(teamName)}">
<input type="hidden" name="app_token" value="${escapeHtml(env.SLACK_APP_TOKEN)}">
<input type="hidden" name="state" value="${escapeHtml(csrfState)}">
<noscript><button type="submit">Click to complete authorization</button></noscript>
</form>
<script>document.getElementById('f').submit();</script>
</body></html>`;

	return new Response(html, {
		headers: { "Content-Type": "text/html; charset=utf-8" },
	});
}

function errorPage(message: string): Response {
	const html = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Slacko — Error</title></head>
<body>
<h2>Authorization failed</h2>
<p>${escapeHtml(message)}</p>
<p>You can close this window and try again.</p>
</body></html>`;

	return new Response(html, {
		status: 400,
		headers: { "Content-Type": "text/html; charset=utf-8" },
	});
}

function escapeHtml(s: string): string {
	return s
		.replace(/&/g, "&amp;")
		.replace(/</g, "&lt;")
		.replace(/>/g, "&gt;")
		.replace(/"/g, "&quot;")
		.replace(/'/g, "&#39;");
}
