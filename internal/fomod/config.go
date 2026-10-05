package fomod

// Load parses ModuleConfig.xml at path.
func Load(path string) (Config, error) {
	b, err := readXMLBytes(path)
	if err != nil {
		return Config{}, err
	}
	return Parse(b)
}

// Open finds and parses a config under root. ok is false when none is there.
func Open(root string) (Config, string, bool, error) {
	path, err := FindConfig(root)
	if err != nil || path == "" {
		return Config{}, "", false, err
	}
	cfg, err := Load(path)
	return cfg, path, err == nil, err
}
