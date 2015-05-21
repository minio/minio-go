package objectstorage_test

import (
	"fmt"
	"testing"

	"github.com/minio/objectstorage-go"
)

func ExampleGetPartSize(t *testing.T) {
	fmt.Println(objectstorage.GetPartSize(5000000000000000000))
}
