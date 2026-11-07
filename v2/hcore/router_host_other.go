//go:build !linux || android

package hcore

// Router mode runs only on Linux; its config fails to build elsewhere (config.forRouterMode).

func prepareRouterHost() {}

func restoreRouterHost() {}
