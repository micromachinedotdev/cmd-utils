package utils

func HasAssets(conf *WranglerConfig) bool {
	return conf.Assets != nil
}
