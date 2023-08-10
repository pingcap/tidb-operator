package filereader

type Config struct {
	Files           string
	Type            string
	RateLimit       float64
	NWorkers        int
	Direct          bool
	CheckpointEvery uint64
	CheckpointFile  string
}
