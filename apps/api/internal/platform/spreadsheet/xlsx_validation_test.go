package spreadsheet

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateXLSXZipRejectsInvalidZip(t *testing.T) {
	err := ValidateXLSXZip([]byte("not a zip"))

	require.Error(t, err)
}

func TestValidateXLSXZipRejectsTooManyEntries(t *testing.T) {
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for i := 0; i < MaxXLSXZipEntries+1; i++ {
		_, err := writer.CreateHeader(&zip.FileHeader{Name: "xl/worksheets/sheet" + string(rune('a'+i%26)) + ".xml"})
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	err := ValidateXLSXZip(buf.Bytes())

	require.ErrorContains(t, err, "項目數超過上限")
}
