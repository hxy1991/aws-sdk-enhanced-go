package appconfig

import (
	"time"

	"github.com/hxy1991/aws-sdk-enhanced-go/awsenhanced/logger"
)

type Option interface {
	apply(*EnhancedAppConfig) error
}

type optionFunc func(*EnhancedAppConfig) error

func (f optionFunc) apply(appConfig *EnhancedAppConfig) error {
	return f(appConfig)
}

func WithRegionName(regionName string) Option {
	return optionFunc(func(appConfig *EnhancedAppConfig) error {
		appConfig.regionName = regionName

		err := appConfig.initAppConfigClient()
		if err != nil {
			return err
		}

		return nil
	})
}

func WithApplicationName(applicationName string) Option {
	return optionFunc(func(appConfig *EnhancedAppConfig) error {
		appConfig.applicationName = applicationName
		return nil
	})
}

func WithEnvironmentName(environmentName string) Option {
	return optionFunc(func(appConfig *EnhancedAppConfig) error {
		appConfig.environmentName = environmentName
		return nil
	})
}

func WithClientId(clientId string) Option {
	return optionFunc(func(appConfig *EnhancedAppConfig) error {
		appConfig.clientId = clientId
		return nil
	})
}

func WithIsCache(isCache bool) Option {
	return optionFunc(func(appConfig *EnhancedAppConfig) error {
		appConfig.isCache = isCache

		if appConfig.cache != nil {
			if !isCache {
				// 原先开启缓存，现在关闭缓存
				appConfig.cache = nil
				appConfig.cacheRefreshTicker.Stop()
				appConfig.cacheRefreshTicker = nil
				logger.Warn("cacheRefreshTicker has been stopped and cache has been shut down")
			}
		} else {
			if isCache {
				// 原先没开启缓存，现在开启缓存
				if appConfig.cacheLimit == 0 {
					appConfig.cacheLimit = defaultCacheLimit
				}

				if appConfig.cacheRefreshInterval == 0 {
					appConfig.cacheRefreshInterval = defaultCacheRefreshInterval
				}

				appConfig.initCache()
			}
		}
		return nil
	})
}

func WithCacheLimit(cacheLimit int64) Option {
	return optionFunc(func(appConfig *EnhancedAppConfig) error {
		appConfig.cacheLimit = cacheLimit

		if appConfig.cache != nil {
			if cacheLimit != 0 {
				oldCacheLimit := appConfig.cache.UpdateCacheLimit(cacheLimit)
				logger.Warn("reset cacheLimit from ", oldCacheLimit, " to ", cacheLimit)
			}
		}
		return nil
	})
}

func WithCacheRefreshInterval(cacheRefreshInterval time.Duration) Option {
	return optionFunc(func(appConfig *EnhancedAppConfig) error {
		appConfig.cacheRefreshInterval = cacheRefreshInterval

		if appConfig.cache != nil {
			if cacheRefreshInterval != 0 {
				oldInterval := appConfig.cacheRefreshTicker.Reset(cacheRefreshInterval)
				logger.Warn("reset refresh cache ticker interval from ", oldInterval, " to ", cacheRefreshInterval)
			}
		}
		return nil
	})
}

func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(appConfig *EnhancedAppConfig) error {
		oldTime := appConfig.timeout
		appConfig.timeout = timeout

		if oldTime != 0 {
			logger.Info("reset timeout from ", oldTime, " to ", timeout)
		}

		return nil
	})
}

func WithXRayEnable(isXRayEnable bool) Option {
	return optionFunc(func(appConfig *EnhancedAppConfig) error {
		appConfig.isXRayEnable = isXRayEnable

		err := appConfig.initAppConfigClient()
		if err != nil {
			return err
		}

		return nil
	})
}
