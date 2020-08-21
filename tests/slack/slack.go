// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"k8s.io/klog"
)

var (
	TestName     string
	Channel      string
	WebhookURL   string
	SuccessCount int
)

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type Attachment struct {
	Fallback   string   `json:"fallback"`
	Color      string   `json:"color"`
	PreText    string   `json:"pretext"`
	AuthorName string   `json:"author_name"`
	AuthorLink string   `json:"author_link"`
	AuthorIcon string   `json:"author_icon"`
	Title      string   `json:"title"`
	TitleLink  string   `json:"title_link"`
	Text       string   `json:"text"`
	ImageURL   string   `json:"image_url"`
	Fields     []*Field `json:"fields"`
	Footer     string   `json:"footer"`
	FooterIcon string   `json:"footer_icon"`
	Timestamp  int64    `json:"ts"`
	MarkdownIn []string `json:"mrkdwn_in"`
}

type Payload struct {
	Parse       string       `json:"parse,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	LinkNames   string       `json:"link_names,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	UnfurlLinks bool         `json:"unfurl_links,omitempty"`
	UnfurlMedia bool         `json:"unfurl_media,omitempty"`
}

func (attachment *Attachment) AddField(field Field) *Attachment {
	attachment.Fields = append(attachment.Fields, &field)
	return attachment
}

func Send(webhookURL string, proxy string, payload Payload) error {
	if webhookURL == "" {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("Error sending msg %+v. Status: %v", payload, resp.Status)
	}
	return nil
}

func SendErrMsg(msg string) error {
	attachment := Attachment{
		Title: "operator stability test failed",
		Color: "fatal",
	}
	payload := Payload{
		Username:    TestName,
		Channel:     Channel,
		Text:        msg,
		IconEmoji:   ":ghost:",
		Attachments: []Attachment{attachment},
	}
	err := Send(WebhookURL, "", payload)
	if err != nil {
		return err
	}
	return nil
}

func SendGoodMsg(msg string) error {
	attachment := Attachment{
		Title: "operator stability test succeeded",
		Color: "good",
	}
	payload := Payload{
		Username:    TestName,
		Channel:     Channel,
		Text:        msg,
		IconEmoji:   ":sun_with_face:",
		Attachments: []Attachment{attachment},
	}
	err := Send(WebhookURL, "", payload)
	if err != nil {
		return err
	}

	return nil
}

func SendWarnMsg(msg string) error {
	attachment := Attachment{
		Title: "operator stability test happen warning",
		Color: "warning",
	}
	payload := Payload{
		Username:    TestName,
		Channel:     Channel,
		Text:        msg,
		IconEmoji:   ":imp:",
		Attachments: []Attachment{attachment},
	}
	err := Send(WebhookURL, "", payload)
	if err != nil {
		return err
	}
	return nil
}

func NotifyAndPanic(err error) {
	sendErr := SendErrMsg(fmt.Sprintf("Succeed %d times, then failed: %s", SuccessCount, err.Error()))
	if sendErr != nil {
		klog.Warningf("failed to notify slack[%s] the massage: %v,error: %v", WebhookURL, err, sendErr)
	}
	time.Sleep(3 * time.Second)
	panic(err)
}

func NotifyAndCompletedf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	sendErr := SendGoodMsg(msg)
	if sendErr != nil {
		klog.Warningf("failed to notify slack[%s] the massage: %s,error: %v", WebhookURL, msg, sendErr)
	}
	klog.Infof(msg)
}
