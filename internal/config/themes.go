package config

// BuiltinTheme returns a fully populated Theme for the given preset name.
// Unknown names fall back to "default".
func BuiltinTheme(name string) Theme {
	switch name {
	case "dark":
		return darkTheme()
	case "light":
		return lightTheme()
	case "monokai":
		return monokaiTheme()
	case "solarized_dark":
		return solarizedDarkTheme()
	case "solarized_light":
		return solarizedLightTheme()
	default:
		return defaultTheme()
	}
}

// defaultTheme matches the original hardcoded colors exactly.
func defaultTheme() Theme {
	return Theme{
		Preset: "default",
		Border: BorderTheme{
			Focused: makeStyle("blue", "", ""),
			Normal:  makeStyle("gray", "", ""),
		},
		Title: TitleTheme{
			Focused: makeStyle("white", "", "b"),
			Normal:  makeStyle("gray", "", ""),
		},
		ChannelsTree: ChannelsTreeTheme{
			Channel:  makeStyle("white", "", ""),
			Selected: makeStyle("blue", "", "b"),
			Unread:   makeStyle("white", "", "b"),
		},
		MessagesList: MessagesListTheme{
			Message:         makeStyle("white", "", ""),
			Author:          makeStyle("green", "", "b"),
			Timestamp:       makeStyle("gray", "", ""),
			Selected:        makeStyle("white", "", "r"),
			Reply:           makeStyle("cyan", "", ""),
			SystemMessage:   makeStyle("gray", "", "d"),
			EditedIndicator: makeStyle("gray", "", "d"),
			PinIndicator:    makeStyle("yellow", "", ""),
			FileAttachment:  makeStyle("blue", "", ""),
			ReactionSelf:    makeStyle("yellow", "", ""),
			ReactionOther:   makeStyle("gray", "", ""),
			DateSeparator:   makeStyle("gray", "", ""),
			NewMsgSeparator: makeStyle("red", "", ""),
		},
		MessageInput: MessageInputTheme{
			Text:        makeStyle("white", "", ""),
			Placeholder: makeStyle("gray", "", "d"),
		},
		ThreadView: ThreadViewTheme{
			Author:          makeStyle("green", "", "b"),
			Timestamp:       makeStyle("gray", "", ""),
			ParentLabel:     makeStyle("gray", "", ""),
			Separator:       makeStyle("gray", "", ""),
			EditedIndicator: makeStyle("gray", "", "d"),
			FileAttachment:  makeStyle("blue", "", ""),
			Reaction:        makeStyle("gray", "", ""),
		},
		Markdown: MarkdownTheme{
			UserMention:    makeStyle("yellow", "", "b"),
			ChannelMention: makeStyle("cyan", "", "b"),
			SpecialMention: makeStyle("yellow", "", "bu"),
			Link:           makeStyle("blue", "", "u"),
			InlineCode:     makeStyle("gray", "", ""),
			CodeFence:      makeStyle("gray", "", ""),
			BlockquoteMark: makeStyle("gray", "", ""),
			BlockquoteText: makeStyle("", "", "d"),
		},
		Modal: ModalTheme{
			InputBackground: makeStyle("", "", ""),
			SecondaryText:   makeStyle("gray", "", ""),
		},
		StatusBar: StatusBarTheme{
			Text:       makeStyle("white", "", ""),
			Background: makeStyle("", "blue", ""),
		},
	}
}

// darkTheme is a high-contrast dark variant.
func darkTheme() Theme {
	return Theme{
		Preset: "dark",
		Border: BorderTheme{
			Focused: makeStyle("#5f87ff", "", ""),
			Normal:  makeStyle("#585858", "", ""),
		},
		Title: TitleTheme{
			Focused: makeStyle("#eeeeee", "", "b"),
			Normal:  makeStyle("#585858", "", ""),
		},
		ChannelsTree: ChannelsTreeTheme{
			Channel:  makeStyle("#bcbcbc", "", ""),
			Selected: makeStyle("#5f87ff", "", "b"),
			Unread:   makeStyle("#eeeeee", "", "b"),
		},
		MessagesList: MessagesListTheme{
			Message:         makeStyle("#eeeeee", "", ""),
			Author:          makeStyle("#5faf5f", "", "b"),
			Timestamp:       makeStyle("#585858", "", ""),
			Selected:        makeStyle("#eeeeee", "", "r"),
			Reply:           makeStyle("#5fafd7", "", ""),
			SystemMessage:   makeStyle("#585858", "", "d"),
			EditedIndicator: makeStyle("#585858", "", "d"),
			PinIndicator:    makeStyle("#d7af5f", "", ""),
			FileAttachment:  makeStyle("#5f87ff", "", ""),
			ReactionSelf:    makeStyle("#d7af5f", "", ""),
			ReactionOther:   makeStyle("#585858", "", ""),
			DateSeparator:   makeStyle("#585858", "", ""),
			NewMsgSeparator: makeStyle("#d75f5f", "", ""),
		},
		MessageInput: MessageInputTheme{
			Text:        makeStyle("#eeeeee", "", ""),
			Placeholder: makeStyle("#585858", "", "d"),
		},
		ThreadView: ThreadViewTheme{
			Author:          makeStyle("#5faf5f", "", "b"),
			Timestamp:       makeStyle("#585858", "", ""),
			ParentLabel:     makeStyle("#585858", "", ""),
			Separator:       makeStyle("#585858", "", ""),
			EditedIndicator: makeStyle("#585858", "", "d"),
			FileAttachment:  makeStyle("#5f87ff", "", ""),
			Reaction:        makeStyle("#585858", "", ""),
		},
		Markdown: MarkdownTheme{
			UserMention:    makeStyle("#d7af5f", "", "b"),
			ChannelMention: makeStyle("#5fafd7", "", "b"),
			SpecialMention: makeStyle("#d7af5f", "", "bu"),
			Link:           makeStyle("#5f87ff", "", "u"),
			InlineCode:     makeStyle("#8a8a8a", "", ""),
			CodeFence:      makeStyle("#8a8a8a", "", ""),
			BlockquoteMark: makeStyle("#585858", "", ""),
			BlockquoteText: makeStyle("", "", "d"),
		},
		Modal: ModalTheme{
			InputBackground: makeStyle("", "", ""),
			SecondaryText:   makeStyle("#585858", "", ""),
		},
		StatusBar: StatusBarTheme{
			Text:       makeStyle("#eeeeee", "", ""),
			Background: makeStyle("", "#5f87ff", ""),
		},
	}
}

// lightTheme is designed for light terminal backgrounds.
func lightTheme() Theme {
	return Theme{
		Preset: "light",
		Border: BorderTheme{
			Focused: makeStyle("#0087af", "", ""),
			Normal:  makeStyle("#a8a8a8", "", ""),
		},
		Title: TitleTheme{
			Focused: makeStyle("#1c1c1c", "", "b"),
			Normal:  makeStyle("#a8a8a8", "", ""),
		},
		ChannelsTree: ChannelsTreeTheme{
			Channel:  makeStyle("#3a3a3a", "", ""),
			Selected: makeStyle("#0087af", "", "b"),
			Unread:   makeStyle("#1c1c1c", "", "b"),
		},
		MessagesList: MessagesListTheme{
			Message:         makeStyle("#1c1c1c", "", ""),
			Author:          makeStyle("#008700", "", "b"),
			Timestamp:       makeStyle("#a8a8a8", "", ""),
			Selected:        makeStyle("#1c1c1c", "", "r"),
			Reply:           makeStyle("#0087af", "", ""),
			SystemMessage:   makeStyle("#a8a8a8", "", "d"),
			EditedIndicator: makeStyle("#a8a8a8", "", "d"),
			PinIndicator:    makeStyle("#af8700", "", ""),
			FileAttachment:  makeStyle("#0087af", "", ""),
			ReactionSelf:    makeStyle("#af8700", "", ""),
			ReactionOther:   makeStyle("#a8a8a8", "", ""),
			DateSeparator:   makeStyle("#a8a8a8", "", ""),
			NewMsgSeparator: makeStyle("#d70000", "", ""),
		},
		MessageInput: MessageInputTheme{
			Text:        makeStyle("#1c1c1c", "", ""),
			Placeholder: makeStyle("#a8a8a8", "", "d"),
		},
		ThreadView: ThreadViewTheme{
			Author:          makeStyle("#008700", "", "b"),
			Timestamp:       makeStyle("#a8a8a8", "", ""),
			ParentLabel:     makeStyle("#a8a8a8", "", ""),
			Separator:       makeStyle("#a8a8a8", "", ""),
			EditedIndicator: makeStyle("#a8a8a8", "", "d"),
			FileAttachment:  makeStyle("#0087af", "", ""),
			Reaction:        makeStyle("#a8a8a8", "", ""),
		},
		Markdown: MarkdownTheme{
			UserMention:    makeStyle("#af8700", "", "b"),
			ChannelMention: makeStyle("#0087af", "", "b"),
			SpecialMention: makeStyle("#af8700", "", "bu"),
			Link:           makeStyle("#005faf", "", "u"),
			InlineCode:     makeStyle("#585858", "", ""),
			CodeFence:      makeStyle("#585858", "", ""),
			BlockquoteMark: makeStyle("#a8a8a8", "", ""),
			BlockquoteText: makeStyle("", "", "d"),
		},
		Modal: ModalTheme{
			InputBackground: makeStyle("", "", ""),
			SecondaryText:   makeStyle("#a8a8a8", "", ""),
		},
		StatusBar: StatusBarTheme{
			Text:       makeStyle("#1c1c1c", "", ""),
			Background: makeStyle("", "#0087af", ""),
		},
	}
}

// monokaiTheme is a developer-friendly theme inspired by Monokai.
func monokaiTheme() Theme {
	return Theme{
		Preset: "monokai",
		Border: BorderTheme{
			Focused: makeStyle("#66d9ef", "", ""),
			Normal:  makeStyle("#75715e", "", ""),
		},
		Title: TitleTheme{
			Focused: makeStyle("#f8f8f2", "", "b"),
			Normal:  makeStyle("#75715e", "", ""),
		},
		ChannelsTree: ChannelsTreeTheme{
			Channel:  makeStyle("#f8f8f2", "", ""),
			Selected: makeStyle("#66d9ef", "", "b"),
			Unread:   makeStyle("#a6e22e", "", "b"),
		},
		MessagesList: MessagesListTheme{
			Message:         makeStyle("#f8f8f2", "", ""),
			Author:          makeStyle("#a6e22e", "", "b"),
			Timestamp:       makeStyle("#75715e", "", ""),
			Selected:        makeStyle("#f8f8f2", "", "r"),
			Reply:           makeStyle("#66d9ef", "", ""),
			SystemMessage:   makeStyle("#75715e", "", "d"),
			EditedIndicator: makeStyle("#75715e", "", "d"),
			PinIndicator:    makeStyle("#e6db74", "", ""),
			FileAttachment:  makeStyle("#66d9ef", "", ""),
			ReactionSelf:    makeStyle("#e6db74", "", ""),
			ReactionOther:   makeStyle("#75715e", "", ""),
			DateSeparator:   makeStyle("#75715e", "", ""),
			NewMsgSeparator: makeStyle("#f92672", "", ""),
		},
		MessageInput: MessageInputTheme{
			Text:        makeStyle("#f8f8f2", "", ""),
			Placeholder: makeStyle("#75715e", "", "d"),
		},
		ThreadView: ThreadViewTheme{
			Author:          makeStyle("#a6e22e", "", "b"),
			Timestamp:       makeStyle("#75715e", "", ""),
			ParentLabel:     makeStyle("#75715e", "", ""),
			Separator:       makeStyle("#75715e", "", ""),
			EditedIndicator: makeStyle("#75715e", "", "d"),
			FileAttachment:  makeStyle("#66d9ef", "", ""),
			Reaction:        makeStyle("#75715e", "", ""),
		},
		Markdown: MarkdownTheme{
			UserMention:    makeStyle("#e6db74", "", "b"),
			ChannelMention: makeStyle("#66d9ef", "", "b"),
			SpecialMention: makeStyle("#e6db74", "", "bu"),
			Link:           makeStyle("#66d9ef", "", "u"),
			InlineCode:     makeStyle("#75715e", "", ""),
			CodeFence:      makeStyle("#75715e", "", ""),
			BlockquoteMark: makeStyle("#75715e", "", ""),
			BlockquoteText: makeStyle("", "", "d"),
		},
		Modal: ModalTheme{
			InputBackground: makeStyle("", "", ""),
			SecondaryText:   makeStyle("#75715e", "", ""),
		},
		StatusBar: StatusBarTheme{
			Text:       makeStyle("#f8f8f2", "", ""),
			Background: makeStyle("", "#75715e", ""),
		},
	}
}

// solarizedDarkTheme implements the Solarized Dark color scheme.
func solarizedDarkTheme() Theme {
	return Theme{
		Preset: "solarized_dark",
		Border: BorderTheme{
			Focused: makeStyle("#268bd2", "", ""),
			Normal:  makeStyle("#586e75", "", ""),
		},
		Title: TitleTheme{
			Focused: makeStyle("#eee8d5", "", "b"),
			Normal:  makeStyle("#586e75", "", ""),
		},
		ChannelsTree: ChannelsTreeTheme{
			Channel:  makeStyle("#839496", "", ""),
			Selected: makeStyle("#268bd2", "", "b"),
			Unread:   makeStyle("#eee8d5", "", "b"),
		},
		MessagesList: MessagesListTheme{
			Message:         makeStyle("#839496", "", ""),
			Author:          makeStyle("#859900", "", "b"),
			Timestamp:       makeStyle("#586e75", "", ""),
			Selected:        makeStyle("#839496", "", "r"),
			Reply:           makeStyle("#2aa198", "", ""),
			SystemMessage:   makeStyle("#586e75", "", "d"),
			EditedIndicator: makeStyle("#586e75", "", "d"),
			PinIndicator:    makeStyle("#b58900", "", ""),
			FileAttachment:  makeStyle("#268bd2", "", ""),
			ReactionSelf:    makeStyle("#b58900", "", ""),
			ReactionOther:   makeStyle("#586e75", "", ""),
			DateSeparator:   makeStyle("#586e75", "", ""),
			NewMsgSeparator: makeStyle("#dc322f", "", ""),
		},
		MessageInput: MessageInputTheme{
			Text:        makeStyle("#839496", "", ""),
			Placeholder: makeStyle("#586e75", "", "d"),
		},
		ThreadView: ThreadViewTheme{
			Author:          makeStyle("#859900", "", "b"),
			Timestamp:       makeStyle("#586e75", "", ""),
			ParentLabel:     makeStyle("#586e75", "", ""),
			Separator:       makeStyle("#586e75", "", ""),
			EditedIndicator: makeStyle("#586e75", "", "d"),
			FileAttachment:  makeStyle("#268bd2", "", ""),
			Reaction:        makeStyle("#586e75", "", ""),
		},
		Markdown: MarkdownTheme{
			UserMention:    makeStyle("#b58900", "", "b"),
			ChannelMention: makeStyle("#2aa198", "", "b"),
			SpecialMention: makeStyle("#b58900", "", "bu"),
			Link:           makeStyle("#268bd2", "", "u"),
			InlineCode:     makeStyle("#586e75", "", ""),
			CodeFence:      makeStyle("#586e75", "", ""),
			BlockquoteMark: makeStyle("#586e75", "", ""),
			BlockquoteText: makeStyle("", "", "d"),
		},
		Modal: ModalTheme{
			InputBackground: makeStyle("", "", ""),
			SecondaryText:   makeStyle("#586e75", "", ""),
		},
		StatusBar: StatusBarTheme{
			Text:       makeStyle("#eee8d5", "", ""),
			Background: makeStyle("", "#268bd2", ""),
		},
	}
}

// solarizedLightTheme implements the Solarized Light color scheme.
func solarizedLightTheme() Theme {
	return Theme{
		Preset: "solarized_light",
		Border: BorderTheme{
			Focused: makeStyle("#268bd2", "", ""),
			Normal:  makeStyle("#93a1a1", "", ""),
		},
		Title: TitleTheme{
			Focused: makeStyle("#073642", "", "b"),
			Normal:  makeStyle("#93a1a1", "", ""),
		},
		ChannelsTree: ChannelsTreeTheme{
			Channel:  makeStyle("#657b83", "", ""),
			Selected: makeStyle("#268bd2", "", "b"),
			Unread:   makeStyle("#073642", "", "b"),
		},
		MessagesList: MessagesListTheme{
			Message:         makeStyle("#657b83", "", ""),
			Author:          makeStyle("#859900", "", "b"),
			Timestamp:       makeStyle("#93a1a1", "", ""),
			Selected:        makeStyle("#657b83", "", "r"),
			Reply:           makeStyle("#2aa198", "", ""),
			SystemMessage:   makeStyle("#93a1a1", "", "d"),
			EditedIndicator: makeStyle("#93a1a1", "", "d"),
			PinIndicator:    makeStyle("#b58900", "", ""),
			FileAttachment:  makeStyle("#268bd2", "", ""),
			ReactionSelf:    makeStyle("#b58900", "", ""),
			ReactionOther:   makeStyle("#93a1a1", "", ""),
			DateSeparator:   makeStyle("#93a1a1", "", ""),
			NewMsgSeparator: makeStyle("#dc322f", "", ""),
		},
		MessageInput: MessageInputTheme{
			Text:        makeStyle("#657b83", "", ""),
			Placeholder: makeStyle("#93a1a1", "", "d"),
		},
		ThreadView: ThreadViewTheme{
			Author:          makeStyle("#859900", "", "b"),
			Timestamp:       makeStyle("#93a1a1", "", ""),
			ParentLabel:     makeStyle("#93a1a1", "", ""),
			Separator:       makeStyle("#93a1a1", "", ""),
			EditedIndicator: makeStyle("#93a1a1", "", "d"),
			FileAttachment:  makeStyle("#268bd2", "", ""),
			Reaction:        makeStyle("#93a1a1", "", ""),
		},
		Markdown: MarkdownTheme{
			UserMention:    makeStyle("#b58900", "", "b"),
			ChannelMention: makeStyle("#2aa198", "", "b"),
			SpecialMention: makeStyle("#b58900", "", "bu"),
			Link:           makeStyle("#268bd2", "", "u"),
			InlineCode:     makeStyle("#93a1a1", "", ""),
			CodeFence:      makeStyle("#93a1a1", "", ""),
			BlockquoteMark: makeStyle("#93a1a1", "", ""),
			BlockquoteText: makeStyle("", "", "d"),
		},
		Modal: ModalTheme{
			InputBackground: makeStyle("", "", ""),
			SecondaryText:   makeStyle("#93a1a1", "", ""),
		},
		StatusBar: StatusBarTheme{
			Text:       makeStyle("#073642", "", ""),
			Background: makeStyle("", "#eee8d5", ""),
		},
	}
}
