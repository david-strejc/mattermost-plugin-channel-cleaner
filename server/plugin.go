package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Plugin struct {
	plugin.MattermostPlugin
	configurationLock sync.RWMutex
	configuration     *configuration
	
	client    *pluginapi.Client
	botUserID string
}

type configuration struct {
	AllowedRoles          string `json:"AllowedRoles"`
	RequireConfirmation   bool   `json:"RequireConfirmation"`
	ExcludeSystemMessages bool   `json:"ExcludeSystemMessages"`
	LogClearActions       bool   `json:"LogClearActions"`
}

func (p *Plugin) OnActivate() error {
	p.API.LogInfo("Channel Cleaner plugin OnActivate called")
	
	p.client = pluginapi.NewClient(p.API, p.Driver)
	
	// Create bot user
	bot := &model.Bot{
		Username:    "channelcleaner",
		DisplayName: "Channel Cleaner",
		Description: "Bot for channel clearing plugin",
	}
	
	botUserID, err := p.client.Bot.EnsureBot(bot)
	if err != nil {
		p.API.LogError("Failed to ensure bot", "error", err.Error())
		return fmt.Errorf("failed to ensure bot: %w", err)
	}
	p.botUserID = botUserID
	p.API.LogInfo("Bot created successfully", "bot_user_id", botUserID)
	
	if err := p.registerCommands(); err != nil {
		return err
	}
	
	p.API.LogInfo("Channel Cleaner plugin activated successfully")
	return nil
}

func (p *Plugin) registerCommands() error {
	p.API.LogInfo("Registering clearchannel command")
	
	command := &model.Command{
		Trigger:          "clearchannel",
		DisplayName:      "Clear Channel",
		Description:      "Clear all messages in the current channel",
		AutoComplete:     true,
		AutoCompleteDesc: "Clear all messages in the current channel",
		AutoCompleteHint: "[confirm]",
	}
	
	if err := p.API.RegisterCommand(command); err != nil {
		p.API.LogError("Failed to register command", "error", err.Error())
		return err
	}
	
	p.API.LogInfo("Successfully registered clearchannel command")
	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	p.API.LogInfo("ExecuteCommand called", "command", args.Command, "user_id", args.UserId, "channel_id", args.ChannelId)
	
	trigger := strings.TrimSpace(strings.Split(args.Command, " ")[0])
	
	if trigger != "/clearchannel" {
		return &model.CommandResponse{}, nil
	}
	
	config := p.getConfiguration()
	
	// Check permissions
	if !p.userHasPermission(args.UserId, args.ChannelId, config.AllowedRoles) {
		return p.responsef("You don't have permission to clear this channel."), nil
	}
	
	// Check for confirmation if required
	commandParts := strings.Fields(args.Command)
	hasConfirm := len(commandParts) > 1 && commandParts[1] == "confirm"
	
	if config.RequireConfirmation && !hasConfirm {
		return p.responsef("⚠️ **Warning**: This will delete all messages in this channel!\n\nTo confirm, type `/clearchannel confirm`"), nil
	}
	
	// Get channel info
	channel, appErr := p.API.GetChannel(args.ChannelId)
	if appErr != nil {
		return p.responsef("Failed to get channel information: %v", appErr), nil
	}
	
	// Clear the channel
	deletedCount, err := p.clearChannel(args.ChannelId, config.ExcludeSystemMessages)
	if err != nil {
		return p.responsef("Failed to clear channel: %v", err), nil
	}
	
	// Log the action if configured
	if config.LogClearActions {
		user, _ := p.API.GetUser(args.UserId)
		p.API.LogInfo(
			"Channel cleared",
			"user_id", args.UserId,
			"username", user.Username,
			"channel_id", args.ChannelId,
			"channel_name", channel.Name,
			"deleted_count", deletedCount,
		)
	}
	
	// Post a system message about the clear action
	user, _ := p.API.GetUser(args.UserId)
	username := "unknown"
	if user != nil {
		username = user.Username
	}
	
	p.API.CreatePost(&model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   fmt.Sprintf("Channel was cleared by @%s. %d messages were deleted.", username, deletedCount),
		Type:      model.PostTypeDefault,
	})
	
	return p.responsef("✅ Successfully cleared %d messages from this channel.", deletedCount), nil
}

func (p *Plugin) clearChannel(channelID string, excludeSystemMessages bool) (int, error) {
	var deletedCount int
	page := 0
	perPage := 200
	
	for {
		posts, err := p.API.GetPostsForChannel(channelID, page, perPage)
		if err != nil {
			return deletedCount, fmt.Errorf("failed to get posts: %w", err)
		}
		
		if len(posts.Order) == 0 {
			break
		}
		
		for _, postID := range posts.Order {
			post := posts.Posts[postID]
			
			// Skip system messages if configured
			if excludeSystemMessages && post.Type != model.PostTypeDefault {
				continue
			}
			
			if err := p.API.DeletePost(postID); err != nil {
				p.API.LogError("Failed to delete post", "post_id", postID, "error", err)
				continue
			}
			
			deletedCount++
		}
		
		page++
	}
	
	return deletedCount, nil
}

func (p *Plugin) userHasPermission(userID, channelID, allowedRoles string) bool {
	// Get user
	user, err := p.API.GetUser(userID)
	if err != nil {
		return false
	}
	
	// Check system admin
	if user.IsSystemAdmin() {
		return true
	}
	
	if allowedRoles == "system_admin" {
		return false
	}
	
	// Check channel admin
	member, err := p.API.GetChannelMember(channelID, userID)
	if err != nil {
		return false
	}
	
	if allowedRoles == "channel_admin" {
		return member.SchemeAdmin
	}
	
	// Allow all members
	return true
}

func (p *Plugin) responsef(format string, args ...interface{}) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         fmt.Sprintf(format, args...),
		Type:         model.PostTypeDefault,
	}
}

func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()
	
	if p.configuration == nil {
		return &configuration{
			AllowedRoles:          "system_admin",
			RequireConfirmation:   true,
			ExcludeSystemMessages: false,
			LogClearActions:       true,
		}
	}
	
	return p.configuration
}

func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()
	
	p.configuration = configuration
}

func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)
	
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return err
	}
	
	p.setConfiguration(configuration)
	
	return nil
}