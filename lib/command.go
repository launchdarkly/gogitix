package lib

type Command struct {
	Command       string `yaml:"command"`
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	ExpectSilence bool   `yaml:"expect_silence"`
	Number        int    `yaml:"-"`
	Path          string `yaml:"-"`
}
