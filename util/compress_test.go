package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompress(t *testing.T) {
	b, err := Compress([]byte("foobar"))
	assert.Nil(t, err)
	assert.NotEmpty(t, b)
}

func TestUncompress(t *testing.T) {
	b, err := Uncompress([]byte("foobar"))
	assert.Error(t, err)

	c, err := Compress([]byte("foobar"))
	assert.Nil(t, err)

	b, err = Uncompress(c.Bytes())
	assert.Nil(t, err)
	assert.NotEmpty(t, b)
}
