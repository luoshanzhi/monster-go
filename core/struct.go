package core

type Setting struct {
	Version   string
	Author    string
	Name      string
	Env       string
	EnvConfig map[string]EnvConfig
}

type EnvConfig struct {
	LogLevel string
	Mysql    MysqlSetting
	Redis    []RedisSettingItem
}

type MysqlSetting struct {
	Master []MysqlSettingItem
	Slave  []MysqlSettingItem
}

type MysqlSettingItem struct {
	Host     string
	User     string
	Password string
	Database string
	Port     int
}

type RedisSettingItem struct {
	Host     string
	Password string
	Port     int
}
