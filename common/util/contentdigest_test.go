package util

import (
	"testing"
	"google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"github.com/stretchr/testify/assert"
)

func TestResourcePathToContentDigest(t *testing.T) {
	for _, tcase := range []struct {
		input string
		isErr bool
		output *remoteexecution.Digest
	} {
		{
			input: "with_instance/blobs/A0F4BBBB11114444/123456789",
			output: &remoteexecution.Digest{Hash: "A0F4BBBB11114444", SizeBytes: 123456789},
		},
		{
			input: "with_instance/uploads/blobs/A0F4BBBB11114444/123456789",
			output: &remoteexecution.Digest{Hash: "A0F4BBBB11114444", SizeBytes: 123456789},
		},
		{
			input: "with_instance/uploads/blobs/A0F4BBBB11114444/123456789/mydir/myfile.zip",
			output: &remoteexecution.Digest{Hash: "A0F4BBBB11114444", SizeBytes: 123456789},
		},
		{
			input: "uploads/blobs/A0F4BBBB11114444/123456789/mydir/myfile.zip",
			output: &remoteexecution.Digest{Hash: "A0F4BBBB11114444", SizeBytes: 123456789},
		},
		{
			input: "blobs/A0F4BBBB11114444/123456789",
			output: &remoteexecution.Digest{Hash: "A0F4BBBB11114444", SizeBytes: 123456789},
		},
		{
			input: "blob/A0F4BBBB11114444/123456789",
			isErr: true,
		},
		{
			input: "blobs/A0F4BBBB11114444/asda",
			isErr: true,
		},
	} {

		t.Run(tcase.input, func(t *testing.T) {
			out, err := ResourcePathToContentDigest(tcase.input)
			if tcase.isErr {
				assert.Error(t, err, "should return an error")
			} else {
				assert.EqualValues(t, tcase.output, out, "should be equal in values")
			}

		})
	}
}

