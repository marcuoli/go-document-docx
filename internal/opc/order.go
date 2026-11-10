package opc

import "github.com/beevik/etree"

// ParagraphOrder is the Word paragraph-property sequence used for inserted nodes.
// Callers treat these internal lookup slices as immutable.
var ParagraphOrder = []string{"pStyle", "keepNext", "keepLines", "pageBreakBefore", "framePr", "widowControl", "numPr", "suppressLineNumbers", "pBdr", "shd", "tabs", "suppressAutoHyphens", "kinsoku", "wordWrap", "overflowPunct", "topLinePunct", "autoSpaceDE", "autoSpaceDN", "bidi", "adjustRightInd", "snapToGrid", "spacing", "ind", "contextualSpacing", "mirrorIndents", "suppressOverlap", "jc", "textDirection", "textAlignment", "textboxTightWrap", "outlineLvl", "divId", "cnfStyle", "rPr", "sectPr", "pPrChange"}

// TableOrder is the Word table-property sequence.
var TableOrder = []string{"tblStyle", "tblpPr", "tblOverlap", "bidiVisual", "tblStyleRowBandSize", "tblStyleColBandSize", "tblW", "jc", "tblCellSpacing", "tblInd", "tblBorders", "shd", "tblLayout", "tblCellMar", "tblLook", "tblCaption", "tblDescription", "tblPrChange"}

// CellOrder is the Word cell-property sequence.
var CellOrder = []string{"cnfStyle", "tcW", "gridSpan", "hMerge", "vMerge", "tcBorders", "shd", "noWrap", "tcMar", "textDirection", "tcFitText", "vAlign", "hideMark", "headers", "cellIns", "cellDel", "cellMerge", "tcPrChange"}

// StyleOrder places paragraph properties before run/table properties in a style.
var StyleOrder = []string{"name", "aliases", "basedOn", "next", "link", "autoRedefine", "hidden", "uiPriority", "semiHidden", "unhideWhenUsed", "qFormat", "locked", "personal", "personalCompose", "personalReply", "rsid", "pPr", "rPr", "tblPr", "trPr", "tcPr", "tblStylePr"}

// RunOrder retains the usual run-property ordering for explicit overrides.
var RunOrder = []string{"rStyle", "rFonts", "b", "bCs", "i", "iCs", "caps", "smallCaps", "strike", "dstrike", "outline", "shadow", "emboss", "imprint", "noProof", "snapToGrid", "vanish", "webHidden", "color", "spacing", "w", "kern", "position", "sz", "szCs", "highlight", "u", "effect", "bdr", "shd", "fitText", "vertAlign", "rtl", "cs", "em", "lang", "eastAsianLayout", "specVanish", "oMath", "rPrChange"}

// Ordered returns an existing Word child or inserts a new one before successors.
// The new node carries its own namespace binding so ancestor-local aliases work.
func Ordered(parent *etree.Element, tag string, order []string) *etree.Element {
	if found := Child(parent, Word, tag); found != nil {
		return found
	}
	e := etree.NewElement("w:" + tag)
	e.CreateAttr("xmlns:w", Word)
	position := len(order)
	for i, t := range order {
		if t == tag {
			position = i
			break
		}
	}
	for _, child := range parent.ChildElements() {
		if child.NamespaceURI() != Word {
			continue
		}
		for i, t := range order {
			if child.Tag == t && i > position {
				parent.InsertChildAt(child.Index(), e)
				return e
			}
		}
	}
	parent.AddChild(e)
	return e
}

// Properties returns a property's container, inserting it first when absent.
func Properties(parent *etree.Element, tag string) *etree.Element {
	if p := Child(parent, Word, tag); p != nil {
		return p
	}
	p := etree.NewElement("w:" + tag)
	p.CreateAttr("xmlns:w", Word)
	parent.InsertChildAt(0, p)
	return p
}
