package navigation

import "testing"

func TestCellScreenTipBookmarkNames(t *testing.T) {
	if name := CellScreenTipMarker(27); name != "_DocxCellTipMark27" || !IsCellScreenTipMarker(name) || IsCellScreenTipMarker("_DocxCellTip27") {
		t.Fatal(name)
	}
	if name := CellScreenTipBookmark(27); name != "_DocxCellTip27" || !IsCellScreenTipBookmark(name) {
		t.Fatal(name)
	}
	for _, name := range []string{"_DocxCellTip6", "_docxcelltip6"} {
		if !IsCellScreenTipBookmark(name) {
			t.Errorf("did not recognize %q", name)
		}
	}
	for _, name := range []string{"authored_tip", "DocxCellTip6", "_DocxCellTipMark6", "DocxCellTip", "DocxCellTip0", "DocxCellTip02", "DocxCellTip2_1", "DocxCellTip1notes", "cell_value_2", "_DocxCellTip２"} {
		if IsCellScreenTipBookmark(name) {
			t.Errorf("misclassified %q", name)
		}
	}
}
