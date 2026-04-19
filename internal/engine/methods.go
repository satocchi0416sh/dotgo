package engine

// GetRootDir returns the root directory of the dotfiles repository
func (e *Engine) GetRootDir() string {
	return e.configMgr.GetRootDir()
}