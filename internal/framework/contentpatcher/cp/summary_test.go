package cp

import "testing"

func TestShapeSummarySkipsPairsThatShareNothing(t *testing.T) {
	prop := func(key string) Shape { return Shape{Kind: 'p', Key: key} }
	sum := func(shapes ...Shape) shapeSummary { return summarize([]*shapeSet{indexShapes(shapes)}) }
	if sum(prop("a")).mayOverlap(sum(prop("b"))) {
		t.Fatal("different property keys cannot overlap")
	}
	if !sum(prop("a"), prop("c")).mayOverlap(sum(prop("c"))) {
		t.Fatal("the same property key may overlap")
	}
	if sum(prop("a")).mayOverlap(sum(Shape{Kind: 'w'})) {
		t.Fatal("a property shape never overlaps a non-property shape")
	}
	if !sum(Shape{Kind: 'w'}).mayOverlap(sum(Shape{Kind: 'w'})) {
		t.Fatal("two non-property shapes may overlap")
	}
}
