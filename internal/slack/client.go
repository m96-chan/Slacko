package slack

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

// Client is a thin wrapper around slack.Client with rate-limit retry
// and cached identity information.
type Client struct {
	api      *slack.Client
	token    string
	UserID   string
	TeamID   string
	TeamName string
	UserName string
}

// New creates a Client, validates the tokens via AuthTest, and populates
// the identity fields.
func New(userToken, appToken string) (*Client, error) {
	if !strings.HasPrefix(appToken, "xapp-") {
		return nil, fmt.Errorf("app token must start with xapp- (got %s...)", safePrefix(appToken))
	}

	api := slack.New(userToken, slack.OptionAppLevelToken(appToken))

	var resp *slack.AuthTestResponse
	err := retryOnRateLimit(func() error {
		var e error
		resp, e = api.AuthTest()
		return e
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		api:      api,
		token:    userToken,
		UserID:   resp.UserID,
		TeamID:   resp.TeamID,
		TeamName: resp.Team,
		UserName: resp.User,
	}, nil
}

// API returns the underlying slack.Client for direct access (e.g. socketmode).
func (c *Client) API() *slack.Client { return c.api }

// Token returns the user token for authenticated HTTP requests.
func (c *Client) Token() string { return c.token }

// retryOnRateLimit executes fn and, if a RateLimitedError is returned,
// sleeps for the requested duration and retries once.
func retryOnRateLimit(fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}

	var rle *slack.RateLimitedError
	if errors.As(err, &rle) {
		time.Sleep(rle.RetryAfter)
		return fn()
	}
	return err
}

// GetConversations returns a page of conversations.
func (c *Client) GetConversations(params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
	var (
		channels []slack.Channel
		cursor   string
	)
	err := retryOnRateLimit(func() error {
		var e error
		channels, cursor, e = c.api.GetConversations(params)
		return e
	})
	return channels, cursor, err
}

// GetConversationHistory returns message history for a conversation.
func (c *Client) GetConversationHistory(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	var resp *slack.GetConversationHistoryResponse
	err := retryOnRateLimit(func() error {
		var e error
		resp, e = c.api.GetConversationHistory(params)
		return e
	})
	return resp, err
}

// GetConversationReplies returns a thread of messages.
func (c *Client) GetConversationReplies(params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
	var (
		msgs    []slack.Message
		hasMore bool
		cursor  string
	)
	err := retryOnRateLimit(func() error {
		var e error
		msgs, hasMore, cursor, e = c.api.GetConversationReplies(params)
		return e
	})
	return msgs, hasMore, cursor, err
}

// PostMessage sends a message to a channel. If the user is not a member of the
// channel, it automatically joins first and retries.
func (c *Client) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	var (
		channel string
		ts      string
	)
	err := retryOnRateLimit(func() error {
		var e error
		channel, ts, e = c.api.PostMessage(channelID, options...)
		return e
	})
	if err != nil && isNotInChannel(err) {
		if _, _, _, joinErr := c.api.JoinConversation(channelID); joinErr != nil {
			return "", "", fmt.Errorf("auto-join failed: %w", joinErr)
		}
		err = retryOnRateLimit(func() error {
			var e error
			channel, ts, e = c.api.PostMessage(channelID, options...)
			return e
		})
	}
	return channel, ts, err
}

// isNotInChannel checks if the error is a "not_in_channel" Slack API error.
func isNotInChannel(err error) bool {
	var slackErr slack.SlackErrorResponse
	if errors.As(err, &slackErr) {
		return slackErr.Err == "not_in_channel"
	}
	return strings.Contains(err.Error(), "not_in_channel")
}

// UpdateMessage updates a message in a channel.
func (c *Client) UpdateMessage(channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error) {
	var (
		channel string
		ts      string
		text    string
	)
	err := retryOnRateLimit(func() error {
		var e error
		channel, ts, text, e = c.api.UpdateMessage(channelID, timestamp, options...)
		return e
	})
	return channel, ts, text, err
}

// DeleteMessage deletes a message from a channel.
func (c *Client) DeleteMessage(channelID, timestamp string) (string, string, error) {
	var (
		channel string
		ts      string
	)
	err := retryOnRateLimit(func() error {
		var e error
		channel, ts, e = c.api.DeleteMessage(channelID, timestamp)
		return e
	})
	return channel, ts, err
}

// AddReaction adds a reaction emoji to an item.
func (c *Client) AddReaction(name string, item slack.ItemRef) error {
	return retryOnRateLimit(func() error {
		return c.api.AddReaction(name, item)
	})
}

// RemoveReaction removes a reaction emoji from an item.
func (c *Client) RemoveReaction(name string, item slack.ItemRef) error {
	return retryOnRateLimit(func() error {
		return c.api.RemoveReaction(name, item)
	})
}

// UploadFile uploads a file to Slack.
func (c *Client) UploadFile(params slack.UploadFileParameters) (*slack.FileSummary, error) {
	var file *slack.FileSummary
	err := retryOnRateLimit(func() error {
		var e error
		file, e = c.api.UploadFile(params)
		return e
	})
	return file, err
}

// GetUserInfo returns detailed information about a user.
func (c *Client) GetUserInfo(userID string) (*slack.User, error) {
	var user *slack.User
	err := retryOnRateLimit(func() error {
		var e error
		user, e = c.api.GetUserInfo(userID)
		return e
	})
	return user, err
}

// GetUsers returns all users in the workspace.
func (c *Client) GetUsers() ([]slack.User, error) {
	var users []slack.User
	err := retryOnRateLimit(func() error {
		var e error
		users, e = c.api.GetUsers()
		return e
	})
	return users, err
}

// MarkConversation sets the read cursor for a conversation to a specific message.
func (c *Client) MarkConversation(channel, ts string) error {
	return retryOnRateLimit(func() error {
		return c.api.MarkConversation(channel, ts)
	})
}

// SearchMessages searches for messages matching a query.
func (c *Client) SearchMessages(query string, params slack.SearchParameters) (*slack.SearchMessages, error) {
	var results *slack.SearchMessages
	err := retryOnRateLimit(func() error {
		var e error
		results, e = c.api.SearchMessages(query, params)
		return e
	})
	return results, err
}

// GetPermalink returns the permalink URL for a message.
func (c *Client) GetPermalink(channelID, timestamp string) (string, error) {
	var permalink string
	err := retryOnRateLimit(func() error {
		var e error
		permalink, e = c.api.GetPermalink(&slack.PermalinkParameters{
			Channel: channelID,
			Ts:      timestamp,
		})
		return e
	})
	return permalink, err
}

// ListPins returns all pinned items in a channel.
func (c *Client) ListPins(channel string) ([]slack.Item, error) {
	var items []slack.Item
	err := retryOnRateLimit(func() error {
		var e error
		items, _, e = c.api.ListPins(channel)
		return e
	})
	return items, err
}

// AddPin pins an item to a channel.
func (c *Client) AddPin(channel string, item slack.ItemRef) error {
	return retryOnRateLimit(func() error {
		return c.api.AddPin(channel, item)
	})
}

// RemovePin unpins an item from a channel.
func (c *Client) RemovePin(channel string, item slack.ItemRef) error {
	return retryOnRateLimit(func() error {
		return c.api.RemovePin(channel, item)
	})
}

// GetConversationInfo returns detailed information about a conversation.
func (c *Client) GetConversationInfo(channelID string) (*slack.Channel, error) {
	var ch *slack.Channel
	err := retryOnRateLimit(func() error {
		var e error
		ch, e = c.api.GetConversationInfo(&slack.GetConversationInfoInput{
			ChannelID:         channelID,
			IncludeNumMembers: true,
		})
		return e
	})
	return ch, err
}

// SetTopic sets the topic for a conversation.
func (c *Client) SetTopic(channelID, topic string) (*slack.Channel, error) {
	var ch *slack.Channel
	err := retryOnRateLimit(func() error {
		var e error
		ch, e = c.api.SetTopicOfConversation(channelID, topic)
		return e
	})
	return ch, err
}

// SetPurpose sets the purpose for a conversation.
func (c *Client) SetPurpose(channelID, purpose string) (*slack.Channel, error) {
	var ch *slack.Channel
	err := retryOnRateLimit(func() error {
		var e error
		ch, e = c.api.SetPurposeOfConversation(channelID, purpose)
		return e
	})
	return ch, err
}

// LeaveConversation leaves a conversation.
func (c *Client) LeaveConversation(channelID string) (bool, error) {
	var notInChannel bool
	err := retryOnRateLimit(func() error {
		var e error
		notInChannel, e = c.api.LeaveConversation(channelID)
		return e
	})
	return notInChannel, err
}

// AddStar stars an item.
func (c *Client) AddStar(channel string, item slack.ItemRef) error {
	return retryOnRateLimit(func() error {
		return c.api.AddStar(channel, item)
	})
}

// RemoveStar removes a star from an item.
func (c *Client) RemoveStar(channel string, item slack.ItemRef) error {
	return retryOnRateLimit(func() error {
		return c.api.RemoveStar(channel, item)
	})
}

// ListAllStars returns all starred items for the current user.
func (c *Client) ListAllStars() ([]slack.Item, error) {
	var items []slack.Item
	err := retryOnRateLimit(func() error {
		var e error
		items, e = c.api.ListAllStars()
		return e
	})
	return items, err
}

// SetUserCustomStatus sets the authenticated user's status emoji and text.
func (c *Client) SetUserCustomStatus(statusText, statusEmoji string) error {
	return retryOnRateLimit(func() error {
		return c.api.SetUserCustomStatus(statusText, statusEmoji, 0)
	})
}

// GetUsersInConversation returns user IDs in a channel with pagination.
func (c *Client) GetUsersInConversation(channelID, cursor string, limit int) ([]string, string, error) {
	var userIDs []string
	var nextCursor string
	err := retryOnRateLimit(func() error {
		var e error
		userIDs, nextCursor, e = c.api.GetUsersInConversation(&slack.GetUsersInConversationParameters{
			ChannelID: channelID,
			Cursor:    cursor,
			Limit:     limit,
		})
		return e
	})
	return userIDs, nextCursor, err
}

// ScheduleMessage schedules a message for future delivery.
// postAt is a Unix timestamp string for when to send the message.
func (c *Client) ScheduleMessage(channelID, postAt, text string) (string, string, error) {
	var respChannel, scheduledID string
	err := retryOnRateLimit(func() error {
		var e error
		respChannel, scheduledID, e = c.api.ScheduleMessage(channelID, postAt,
			slack.MsgOptionText(text, false))
		return e
	})
	return respChannel, scheduledID, err
}

// GetScheduledMessages returns scheduled messages, optionally filtered by channel.
func (c *Client) GetScheduledMessages(channelID string) ([]slack.ScheduledMessage, error) {
	var msgs []slack.ScheduledMessage
	err := retryOnRateLimit(func() error {
		var e error
		msgs, _, e = c.api.GetScheduledMessages(&slack.GetScheduledMessagesParameters{
			Channel: channelID,
		})
		return e
	})
	return msgs, err
}

// DeleteScheduledMessage cancels a scheduled message.
func (c *Client) DeleteScheduledMessage(channelID, scheduledMessageID string) error {
	return retryOnRateLimit(func() error {
		_, err := c.api.DeleteScheduledMessage(&slack.DeleteScheduledMessageParameters{
			Channel:            channelID,
			ScheduledMessageID: scheduledMessageID,
		})
		return err
	})
}

// AddReminder creates a reminder for the current user.
func (c *Client) AddReminder(text, timeStr string) (*slack.Reminder, error) {
	var rem *slack.Reminder
	err := retryOnRateLimit(func() error {
		var e error
		rem, e = c.api.AddUserReminder(c.UserID, text, timeStr)
		return e
	})
	return rem, err
}

// ListReminders returns all active reminders.
func (c *Client) ListReminders() ([]*slack.Reminder, error) {
	var rems []*slack.Reminder
	err := retryOnRateLimit(func() error {
		var e error
		rems, e = c.api.ListReminders()
		return e
	})
	return rems, err
}

// DeleteReminder deletes a reminder by ID.
func (c *Client) DeleteReminder(id string) error {
	return retryOnRateLimit(func() error {
		return c.api.DeleteReminder(id)
	})
}

// CreateConversation creates a new channel (public or private).
func (c *Client) CreateConversation(name string, isPrivate bool) (*slack.Channel, error) {
	var ch *slack.Channel
	err := retryOnRateLimit(func() error {
		var e error
		ch, e = c.api.CreateConversation(slack.CreateConversationParams{
			ChannelName: name,
			IsPrivate:   isPrivate,
		})
		return e
	})
	return ch, err
}

// InviteUsersToConversation invites one or more users to a conversation.
func (c *Client) InviteUsersToConversation(channelID string, userIDs ...string) (*slack.Channel, error) {
	var ch *slack.Channel
	err := retryOnRateLimit(func() error {
		var e error
		ch, e = c.api.InviteUsersToConversation(channelID, userIDs...)
		return e
	})
	return ch, err
}

// OpenConversation opens or creates a direct message or group DM conversation
// with the given user IDs.
func (c *Client) OpenConversation(userIDs []string) (*slack.Channel, error) {
	var ch *slack.Channel
	err := retryOnRateLimit(func() error {
		var e error
		ch, _, _, e = c.api.OpenConversation(&slack.OpenConversationParameters{
			Users: userIDs,
		})
		return e
	})
	return ch, err
}

// ListBookmarks returns all bookmarks for a channel.
func (c *Client) ListBookmarks(channelID string) ([]slack.Bookmark, error) {
	var bookmarks []slack.Bookmark
	err := retryOnRateLimit(func() error {
		var e error
		bookmarks, e = c.api.ListBookmarks(channelID)
		return e
	})
	return bookmarks, err
}

// safePrefix returns the first 10 characters of a token for error messages.
func safePrefix(token string) string {
	if len(token) <= 10 {
		return token
	}
	return token[:10]
}
