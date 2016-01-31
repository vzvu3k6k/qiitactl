package model

import (
	"testing"
	"time"
)

func TestNewPostWithWrongMeta(t *testing.T) {
	_, err := NewPost([]byte(`XXXXXXXX
<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
  TypeScript:
  Docker:
    - 1.9
  Go:
    - 1.4.3
    - 1.5.3
-->
# Main title
## Sub title
Paragraph
`))
	if err == nil {
		t.Errorf("start without meta comment should return error")
	}
}

func TestNewPostWithWrongTag(t *testing.T) {
	_, err := NewPost([]byte(`<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
  TypeScript
  Docker:
    - 1.9
  Go:
    - 1.4.3
    - 1.5.3
-->
# Main title
## Sub title
Paragraph
`))
	if err == nil {
		t.Errorf("should return error with non-object element in tags")
	}
}

func TestNewPostWithWrongTitle(t *testing.T) {
	_, err := NewPost([]byte(`<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
  TypeScript:
  Docker:
    - 1.9
  Go:
    - 1.4.3
    - 1.5.3
-->
## Sub title
# Main title
Paragraph
`))
	if err == nil {
		t.Errorf("should return error with non-object element in tags")
	}
}

func TestNewPostWithCorrectText(t *testing.T) {
	post, err := NewPost([]byte(`<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
  TypeScript:
  Docker:
    - 1.9
  Go:
    - 1.4.3
    - 1.5.3
-->
# Main title
## Sub title
Paragraph
`))
	if err != nil {
		t.Fatal(err)
	}

	if post.Meta.Id != "abcdefghijklmnopqrst" {
		t.Errorf("wrong Id")
	}
	if post.Meta.Url != "http://example.com/mypost" {
		t.Errorf("wrong Url")
	}
	if !post.Meta.CreatedAt.Equal(time.Date(2013, 12, 10, 3, 29, 14, 0, time.UTC)) {
		t.Errorf("wrong CreatedAt")
	}
	if !post.Meta.UpdatedAt.Equal(time.Date(2015, 02, 25, 0, 26, 30, 0, time.UTC)) {
		t.Errorf("wrong UpdatedAt")
	}
	if post.Meta.Private != true {
		t.Errorf("wrong Private")
	}
	if post.Meta.Coediting != false {
		t.Errorf("wrong Coediting")
	}
	if len(post.Meta.Tags) != 3 {
		t.Errorf("wrong Tags length: %d", len(post.Meta.Tags))
	} else {
		for _, tag := range post.Meta.Tags {
			switch tag.Name {
			case "TypeScript":
				if len(tag.Versions) != 0 {
					t.Errorf("wrong Tag with no version: %+v", tag)
				}
			case "Docker":
				if len(tag.Versions) != 1 || tag.Versions[0] != "1.9" {
					t.Errorf("wrong Tag with single version: %+v", tag)
				}
			case "Go":
				if len(tag.Versions) != 2 || tag.Versions[0] != "1.4.3" || tag.Versions[1] != "1.5.3" {
					t.Errorf("wrong Tag with multi versions: %+v", tag)
				}
			}
		}
	}
	if post.Title != "Main title" {
		t.Errorf("wrong Title")
	}
	if post.Body != "## Sub title\nParagraph" {
		t.Errorf("wrong Body: %s", post.Body)
	}
}