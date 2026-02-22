package keyring

import (
	"encoding/json"
	"os"
	"path/filepath"

	gokeyring "github.com/zalando/go-keyring"

	"github.com/m96-chan/Slacko/internal/consts"
)

// Workspace represents a stored workspace entry.
type Workspace struct {
	ID     string `json:"id"`      // Slack team ID
	Name   string `json:"name"`    // team/workspace name
	BotKey string `json:"bot_key"` // keyring key for bot token
	AppKey string `json:"app_key"` // keyring key for app token
}

// WorkspaceTokens holds the resolved tokens for a workspace.
type WorkspaceTokens struct {
	BotToken string
	AppToken string
}

const workspacesFile = "workspaces.json"

// workspacesPath returns the path to the workspaces registry file.
func workspacesPath() string {
	return filepath.Join(consts.CacheDir, workspacesFile)
}

// ListWorkspaces returns all stored workspaces.
func ListWorkspaces() ([]Workspace, error) {
	data, err := os.ReadFile(workspacesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ws []Workspace
	if err := json.Unmarshal(data, &ws); err != nil {
		return nil, err
	}
	return ws, nil
}

// saveWorkspaces writes the workspace list to disk.
func saveWorkspaces(ws []Workspace) error {
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(workspacesPath(), data, 0o600)
}

// AddWorkspace stores a new workspace's tokens and adds it to the registry.
// If a workspace with the same ID already exists, it is updated.
func AddWorkspace(id, name, botToken, appToken string) error {
	ws, err := ListWorkspaces()
	if err != nil {
		ws = nil
	}

	botKey := "bot_" + id
	appKey := "app_" + id

	// Store tokens in keyring.
	if err := gokeyring.Set(consts.Name, botKey, botToken); err != nil {
		return err
	}
	if err := gokeyring.Set(consts.Name, appKey, appToken); err != nil {
		return err
	}

	// Update or add workspace entry.
	found := false
	for i, w := range ws {
		if w.ID == id {
			ws[i].Name = name
			ws[i].BotKey = botKey
			ws[i].AppKey = appKey
			found = true
			break
		}
	}
	if !found {
		ws = append(ws, Workspace{
			ID:     id,
			Name:   name,
			BotKey: botKey,
			AppKey: appKey,
		})
	}

	return saveWorkspaces(ws)
}

// RemoveWorkspace removes a workspace from the registry and deletes its tokens.
func RemoveWorkspace(id string) error {
	ws, err := ListWorkspaces()
	if err != nil {
		return err
	}

	var updated []Workspace
	for _, w := range ws {
		if w.ID == id {
			_ = gokeyring.Delete(consts.Name, w.BotKey)
			_ = gokeyring.Delete(consts.Name, w.AppKey)
			continue
		}
		updated = append(updated, w)
	}

	return saveWorkspaces(updated)
}

// GetWorkspaceTokens retrieves the tokens for a workspace from the keyring.
func GetWorkspaceTokens(w Workspace) (WorkspaceTokens, error) {
	bot, err := gokeyring.Get(consts.Name, w.BotKey)
	if err != nil {
		return WorkspaceTokens{}, err
	}
	app, err := gokeyring.Get(consts.Name, w.AppKey)
	if err != nil {
		return WorkspaceTokens{}, err
	}
	return WorkspaceTokens{BotToken: bot, AppToken: app}, nil
}

// MigrateDefaultWorkspace migrates the legacy single-workspace tokens
// to the multi-workspace registry if needed.
func MigrateDefaultWorkspace(teamID, teamName string) error {
	ws, _ := ListWorkspaces()
	// Check if already migrated.
	for _, w := range ws {
		if w.ID == teamID {
			return nil
		}
	}

	// Read legacy tokens.
	bot, err := GetBotToken()
	if err != nil {
		return err
	}
	app, err := GetAppToken()
	if err != nil {
		return err
	}

	return AddWorkspace(teamID, teamName, bot, app)
}
