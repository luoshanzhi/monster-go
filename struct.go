package monster

type Setting struct {
	Version   string
	Author    string
	Name      string
	Env       string
	EnvConfig map[string]EnvConfig
}

type EnvConfig struct {
	LogLevel string
	Sql      map[string]SqlSetting
	Redis    map[string][]RedisSettingItem
	Mongodb  map[string]MongodbSetting
}

type SqlSetting struct {
	Master []SqlSettingItem
	Slave  []SqlSettingItem
}

type SqlSettingItem struct {
	DriverName string
	Host       string
	User       string
	Password   string
	Database   string
	Port       int
}

type RedisSettingItem struct {
	Host     string
	Password string
	Port     int
}

type MongodbSetting struct {
	Master []MongodbSettingItem
	Slave  []MongodbSettingItem
}

type MongodbSettingItem struct {
	Host     string
	User     string
	Password string
	Database string
	Port     int
}
