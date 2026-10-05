package contentpatcher

import "testing"

func TestShapeSummarySkipsPairsThatShareNothing(t *testing.T) {
	prop := func(key string) cpShape { return cpShape{kind: 'p', key: key} }
	sum := func(shapes ...cpShape) shapeSummary { return summarize([]*shapeSet{indexShapes(shapes)}) }
	if sum(prop("a")).mayOverlap(sum(prop("b"))) {
		t.Fatal("different property keys cannot overlap")
	}
	if !sum(prop("a"), prop("c")).mayOverlap(sum(prop("c"))) {
		t.Fatal("the same property key may overlap")
	}
	if sum(prop("a")).mayOverlap(sum(cpShape{kind: 'w'})) {
		t.Fatal("a property shape never overlaps a non-property shape")
	}
	if !sum(cpShape{kind: 'w'}).mayOverlap(sum(cpShape{kind: 'w'})) {
		t.Fatal("two non-property shapes may overlap")
	}
}
