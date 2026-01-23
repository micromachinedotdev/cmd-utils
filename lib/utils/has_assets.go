package utils

func HasAssets(conf map[string]any) bool {
	if _, ok := conf["assets"]; ok {
		if _, ok := conf["assets"].(map[string]any); ok {
			return true
		}
	}
	return false
}
