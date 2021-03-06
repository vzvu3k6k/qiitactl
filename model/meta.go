package model

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v2"
)

// Meta is meta data of post.
type Meta struct {
	ID        string `json:"id" yaml:"id"`                 // 投稿の一意なID
	URL       string `json:"url" yaml:"url"`               // 投稿のURL
	CreatedAt Time   `json:"created_at" yaml:"created_at"` // データが作成された日時
	UpdatedAt Time   `json:"updated_at" yaml:"updated_at"` // データが最後に更新された日時
	Private   bool   `json:"private" yaml:"private"`       // 限定共有状態かどうかを表すフラグ (Qiita:Teamでは無効)
	Coediting bool   `json:"coediting" yaml:"coediting"`   // この投稿が共同更新状態かどうか (Qiita:Teamでのみ有効)
	Tags      Tags   `json:"tags" yaml:"tags"`             // 投稿に付いたタグ一覧
	Team      *Team  `json:"-"`                            // チーム
}

// Encode marshals meta as YAML.
func (meta Meta) Encode() (out string) {
	o, err := yaml.Marshal(meta)
	if err != nil {
		fmt.Println(err)
	}
	out = string(bytes.TrimSpace(o))
	return
}

// Decode unmarshals encoded meta.
func (meta *Meta) Decode(s string) (err error) {
	return yaml.Unmarshal([]byte(s), meta)
}
