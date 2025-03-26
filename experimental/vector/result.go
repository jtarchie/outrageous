package vector

type Result struct {
	ID        string
	Vector    Vector
	Content   string
	Metadata  Metadata
	CreatedAt string

	Distance   int
	Similarity float32
}
