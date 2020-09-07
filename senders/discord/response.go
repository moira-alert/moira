package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (sender *Sender) getResponse(m *discordgo.MessageCreate, channel *discordgo.Channel) (string, error) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == sender.botUserID {
		return "", nil
	}

	// If the message is "!start" update the channel ID for the user/channel
	if m.Content == "!start" {
		switch channel.Type {
		case discordgo.ChannelTypeDM:
			err := sender.DataBase.SetUsernameID(messenger, "@"+m.Author.Username, channel.ID)
			if err != nil {
				return "", fmt.Errorf("error while setting the channel ID for user: %s", err)
			}
			msg := fmt.Sprintf("Okay, %s, your id is %s", m.Author.Username, channel.ID)
			return msg, nil
		case discordgo.ChannelTypeGuildText:
			err := sender.DataBase.SetUsernameID(messenger, channel.Name, channel.ID)
			if err != nil {
				return "", fmt.Errorf("error while setting the channel ID for text channel: %s", err)
			}
			msg := fmt.Sprintf("Hi, all!\nI will send alerts in this group (%s).", channel.Name)
			return msg, nil
		default:
			return "Unsupported channel type", nil
		}
	}

	return "", nil
}
