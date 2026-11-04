package platform_test

import (
	"testing"

	"github.com/aptlogica/sereni-base/internal/platform"
	"github.com/stretchr/testify/assert"
)

func TestUUIDIDsRoundTrip(t *testing.T) {
	const id = "3f2a9c1e-7b4d-4e8a-9c3b-1d2e3f4a5b6c"
	for _, prefix := range []string{platform.PrefixBase, platform.PrefixTable, platform.PrefixField} {
		encoded := platform.EncodeUUID(prefix, id)
		assert.Equal(t, prefix+"3f2a9c1e7b4d4e8a9c3b1d2e3f4a5b6c", encoded)
		decoded, ok := platform.DecodeUUID(prefix, encoded)
		assert.True(t, ok)
		assert.Equal(t, id, decoded)
	}
	// Upper case UUIDs are encoded in lower case
	assert.Equal(t, "tbl_3f2a9c1e7b4d4e8a9c3b1d2e3f4a5b6c", platform.EncodeUUID(platform.PrefixTable, "3F2A9C1E-7B4D-4E8A-9C3B-1D2E3F4A5B6C"))
}

func TestUUIDIDsRejectMalformedValues(t *testing.T) {
	for _, value := range []string{
		"tbl_3f2a9c1e7b4d4e8a9c3b1d2e3f4a5b6",   // 31 characters
		"tbl_3f2a9c1e7b4d4e8a9c3b1d2e3f4a5b6c0", // 33 characters
		"tbl_3f2a9c1e7b4d4e8a9c3b1d2e3f4a5bzz",  // not hex
		"bse_3f2a9c1e7b4d4e8a9c3b1d2e3f4a5b6c",  // other prefix
		"3f2a9c1e7b4d4e8a9c3b1d2e3f4a5b6c",      // no prefix
		"tbl_",
	} {
		_, ok := platform.DecodeUUID(platform.PrefixTable, value)
		assert.False(t, ok, value)
	}
}

func TestRecordIDs(t *testing.T) {
	assert.Equal(t, "rec_1042", platform.EncodeRecordID(1042))
	id, ok := platform.DecodeRecordID("rec_1042")
	assert.True(t, ok)
	assert.Equal(t, int64(1042), id)

	// Every record has exactly one ID
	for _, value := range []string{"rec_", "rec_0", "rec_007", "rec_-1", "rec_+1", "rec_1.5", "rec_12a", "1042", "REC_1", "rec_99999999999999999999"} {
		_, ok := platform.DecodeRecordID(value)
		assert.False(t, ok, value)
	}
}
