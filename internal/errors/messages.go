package errors

// Common error message formats used throughout the application
const (
	ErrCurrentDir   = "failed to get current directory: %w"
	ErrExpandPath   = "failed to expand path: %w"
	ErrHomeDir      = "failed to get home directory: %w"
	ErrLoadManifest = "failed to load manifest: %w"
	ErrReadFile     = "failed to read file: %w"
	ErrWriteFile    = "failed to write file: %w"
	ErrParseYAML    = "failed to parse YAML: %w"
	ErrMarshalYAML  = "failed to marshal YAML: %w"
	ErrCreateDir    = "failed to create directory: %w"
	ErrStatFile     = "failed to stat file: %w"
	ErrCopyFile     = "failed to copy file: %w"
	ErrCreateLink   = "failed to create symlink: %w"
	ErrRemoveFile   = "failed to remove file: %w"
	ErrMoveFile     = "failed to move file: %w"
)
