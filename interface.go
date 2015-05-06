package objectstorage

// API - object storage API interface
type API interface {
	Get()
	Put()
	List()
}

type api struct {
	*lowLevelAPI
}

const LibraryName = "objectstorage-go"
const LibraryVersion = "0.1"

// Config - main configuration struct used by all to set endpoint, credentials, and other options for requests.
type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	ContentType     string
}

// New - instantiate a new minio api client
func New(config *Config) API {
	return &api{&lowLevelAPI{config}}
}

func (a *api) Get() {
}

func (a *api) Put() {
}

func (a *api) Stat() {
}

func (a *api) List() {
}
