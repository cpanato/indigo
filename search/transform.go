package search

import (
	"regexp"
	"strings"
	"time"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/util"
	"github.com/rivo/uniseg"
)

type ProfileDoc struct {
	DocIndexTs  string   `json:"doc_index_ts"`
	DID         string   `json:"did"`
	RecordCID   string   `json:"record_cid"`
	Handle      string   `json:"handle"`
	DisplayName *string  `json:"display_name,omitempty"`
	Description *string  `json:"description,omitempty"`
	ImgAltText  []string `json:"img_alt_text,omitempty"`
	SelfLabel   []string `json:"self_label,omitempty"`
	Hashtag     []string `json:"hashtag,omitempty"`
	Emoji       []string `json:"emoji,omitempty"`
	HasAvatar   bool     `json:"has_avatar"`
	HasBanner   bool     `json:"has_banner"`
}

type PostDoc struct {
	DocIndexTs      string   `json:"doc_index_ts"`
	DID             string   `json:"did"`
	RecordRkey      string   `json:"record_rkey"`
	RecordCID       string   `json:"record_cid"`
	CreatedAt       string   `json:"created_at"`
	Text            string   `json:"text"`
	LangCode        []string `json:"lang_code,omitempty"`
	LangCodeIso2    []string `json:"lang_code_iso2,omitempty"`
	MentionDID      []string `json:"mention_did,omitempty"`
	LinkURL         []string `json:"link_url,omitempty"`
	EmbedURL        *string  `json:"embed_url,omitempty"`
	EmbedATURI      *string  `json:"embed_aturi,omitempty"`
	ReplyRootATURI  *string  `json:"reply_root_aturi,omitempty"`
	EmbedImgCount   int      `json:"embed_img_count"`
	EmbedImgAltText []string `json:"embed_img_alt_text,omitempty"`
	SelfLabel       []string `json:"self_label,omitempty"`
	Hashtag         []string `json:"hashtag,omitempty"`
	Emoji           []string `json:"emoji,omitempty"`
}

// Returns the search index document ID (`_id`) for this document.
//
// This identifier should be URL safe and not contain a slash ("/").
func (d *ProfileDoc) DocId() string {
	return d.DID
}

// Returns the search index document ID (`_id`) for this document.
//
// This identifier should be URL safe and not contain a slash ("/").
func (d *PostDoc) DocId() string {
	return d.DID + "_" + d.RecordRkey
}

func TransformProfile(profile *appbsky.ActorProfile, ident *identity.Identity, cid string) ProfileDoc {
	// TODO: placeholder for future alt text on profile blobs
	var altText []string
	var hashtags []string
	var emojis []string
	if profile.Description != nil {
		hashtags = parseHashtags(*profile.Description)
		emojis = parseEmojis(*profile.Description)
	}
	var selfLabels []string
	if profile.Labels != nil && profile.Labels.LabelDefs_SelfLabels != nil {
		for _, le := range profile.Labels.LabelDefs_SelfLabels.Values {
			selfLabels = append(selfLabels, le.Val)
		}
	}
	handle := ""
	if !ident.Handle.IsInvalidHandle() {
		handle = ident.Handle.String()
	}
	return ProfileDoc{
		DocIndexTs:  time.Now().UTC().Format(util.ISO8601),
		DID:         ident.DID.String(),
		RecordCID:   cid,
		Handle:      handle,
		DisplayName: profile.DisplayName,
		Description: profile.Description,
		ImgAltText:  altText,
		SelfLabel:   selfLabels,
		Hashtag:     hashtags,
		Emoji:       emojis,
		HasAvatar:   profile.Avatar != nil,
		HasBanner:   profile.Banner != nil,
	}
}

func TransformPost(post *appbsky.FeedPost, ident *identity.Identity, rkey, cid string) PostDoc {
	altText := []string{}
	if post.Embed != nil && post.Embed.EmbedImages != nil {
		for _, img := range post.Embed.EmbedImages.Images {
			if img.Alt != "" {
				altText = append(altText, img.Alt)
			}
		}
	}
	var langCodeIso2 []string
	for _, lang := range post.Langs {
		// TODO: include an actual language code map to go from 3char to 2char
		prefix := strings.SplitN(lang, "-", 2)[0]
		if len(prefix) == 2 {
			langCodeIso2 = append(langCodeIso2, strings.ToLower(prefix))
		}
	}
	var mentionDIDs []string
	var linkURLs []string
	for _, facet := range post.Facets {
		for _, feat := range facet.Features {
			if feat.RichtextFacet_Mention != nil {
				mentionDIDs = append(mentionDIDs, feat.RichtextFacet_Mention.Did)
			}
			if feat.RichtextFacet_Link != nil {
				linkURLs = append(linkURLs, feat.RichtextFacet_Link.Uri)
			}
		}
	}
	var replyRootATURI *string
	if post.Reply != nil {
		replyRootATURI = &(post.Reply.Root.Uri)
	}
	var embedURL *string
	if post.Embed != nil && post.Embed.EmbedExternal != nil {
		embedURL = &post.Embed.EmbedExternal.External.Uri
	}
	var embedATURI *string
	if post.Embed != nil && post.Embed.EmbedRecord != nil {
		embedATURI = &post.Embed.EmbedRecord.Record.Uri
	}
	if post.Embed != nil && post.Embed.EmbedRecordWithMedia != nil {
		embedATURI = &post.Embed.EmbedRecordWithMedia.Record.Record.Uri
	}
	var embedImgCount int = 0
	var embedImgAltText []string
	if post.Embed != nil && post.Embed.EmbedImages != nil {
		embedImgCount = len(post.Embed.EmbedImages.Images)
		for _, img := range post.Embed.EmbedImages.Images {
			if img.Alt != "" {
				embedImgAltText = append(embedImgAltText, img.Alt)
			}
		}
	}
	var selfLabels []string
	if post.Labels != nil && post.Labels.LabelDefs_SelfLabels != nil {
		for _, le := range post.Labels.LabelDefs_SelfLabels.Values {
			selfLabels = append(selfLabels, le.Val)
		}
	}

	return PostDoc{
		DocIndexTs:      time.Now().UTC().Format(util.ISO8601),
		DID:             ident.DID.String(),
		RecordRkey:      rkey,
		RecordCID:       cid,
		CreatedAt:       post.CreatedAt,
		Text:            post.Text,
		LangCode:        post.Langs,
		LangCodeIso2:    langCodeIso2,
		MentionDID:      mentionDIDs,
		LinkURL:         linkURLs,
		EmbedURL:        embedURL,
		EmbedATURI:      embedATURI,
		ReplyRootATURI:  replyRootATURI,
		EmbedImgCount:   embedImgCount,
		EmbedImgAltText: embedImgAltText,
		SelfLabel:       selfLabels,
		Hashtag:         parseHashtags(post.Text),
		Emoji:           parseEmojis(post.Text),
	}
}

func parseHashtags(s string) []string {
	var hashtagRegex = regexp.MustCompile(`\B#([A-Za-z]+)\b`)
	var ret []string = []string{}
	seen := make(map[string]bool)
	for _, m := range hashtagRegex.FindAllStringSubmatch(s, -1) {
		if seen[m[1]] == false {
			ret = append(ret, m[1])
			seen[m[1]] = true
		}
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}

func parseEmojis(s string) []string {
	var ret []string = []string{}
	seen := make(map[string]bool)
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		// check if this grapheme cluster starts with an emoji rune (Unicode codepoint, int32)
		firstRune := gr.Runes()[0]
		if (firstRune >= 0x1F000 && firstRune <= 0x1FFFF) || (firstRune >= 0x2600 && firstRune <= 0x26FF) {
			emoji := gr.Str()
			if seen[emoji] == false {
				ret = append(ret, emoji)
				seen[emoji] = true
			}
		}
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}
