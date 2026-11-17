package source

import "testing"

func TestHTMLConversion(t *testing.T) {
	page := `<h3>Title</h3><p>Fixed &amp; <strong>faster</strong>, see <a href="https://x.test/a">the page</a> or <a href="javascript:x">this</a>.</p>` +
		`<blockquote>quoted</blockquote>` +
		`<ul><li>one</li><li>two<br/>x</li></ul><ol><li>first</li><li>second</li></ol><script>no()</script><pre>a  b</pre>`
	if got, want := HTMLText(page), "Title\n\nFixed & faster, see the page or this.\n\nquoted\n\n• one\n• two\nx\n\n1. first\n2. second\n\na  b"; got != want {
		t.Fatalf("text %q, want %q", got, want)
	}
	want := "### Title\n\nFixed & **faster**, see [the page](https://x.test/a) or this.\n\n> quoted\n\n- one\n- two\nx\n\n1. first\n2. second\n\n```\na  b\n```"
	if got := HTMLMarkdown(page); got != want {
		t.Fatalf("markdown %q, want %q", got, want)
	}
}
