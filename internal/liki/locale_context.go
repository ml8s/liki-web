package liki

import "context"

type chatLocaleKey struct{}

// withChatLocale carries the active UI locale through the service layer. It is
// request-scoped: the locale can change between two messages in one session.
func withChatLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, chatLocaleKey{}, locale)
}

func chatLocaleFrom(ctx context.Context) string {
	locale, _ := ctx.Value(chatLocaleKey{}).(string)
	return locale
}
