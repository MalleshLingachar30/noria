package config

// Preferences are owned by the CLI; frontends only read/write through commands.
type Preferences struct {
	LaunchMenuBar bool `json:"launch_menubar_at_login"`
	StartAtLogin  bool `json:"start_at_login"`
	AutoUpdate    bool `json:"auto_update"`
}

func ReadPreferences(home string) (Preferences, error) {
	c, e := Load(home)
	return Preferences{LaunchMenuBar: c.MenuBarAtLogin, StartAtLogin: c.StartAtLogin, AutoUpdate: c.AutoUpdate}, e
}
func SavePreferences(home string, p Preferences) error {
	return Mutate(home, func(c *Config) error {
		c.MenuBarAtLogin = p.LaunchMenuBar
		c.StartAtLogin = p.StartAtLogin
		c.AutoUpdate = p.AutoUpdate
		return nil
	})
}

// Mutate serializes every config read/modify/write, including background refresh.
func Mutate(home string, fn func(*Config) error) error {
	unlock, e := Lock(home, "config")
	if e != nil {
		return e
	}
	defer unlock()
	c, e := Load(home)
	if e != nil {
		return e
	}
	if e = fn(&c); e != nil {
		return e
	}
	return Save(home, c)
}
