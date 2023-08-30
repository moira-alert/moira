package bluge

import (
	"github.com/blugelabs/bluge"
)

type TriggerIndex struct {
	writer *bluge.Writer
}

func CreateTriggerIndex() (*TriggerIndex, error) {
	config := bluge.InMemoryOnlyConfig()
	writer, err := bluge.OpenWriter(config)
	if err != nil {
		return nil, err
	}
	return &TriggerIndex{
		writer: writer,
	}, nil
}

func (index *TriggerIndex) GetCount() (int64, error) {
	reader, err := index.writer.Reader()
	if err != nil {
		return 0, err
	}
	count, err := reader.Count()
	if err != nil {
		return 0, err
	}
	return int64(count), err
}

func (index *TriggerIndex) Close() error {
	return index.writer.Close()
}
