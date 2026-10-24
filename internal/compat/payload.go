package compat

import (
	"encoding/json"
	"fmt"
)

// pushRequest is the body of POST /api/v2/push/.
type pushRequest struct {
	Device  string `json:"device"`
	Token   string `json:"token"`
	Channel string `json:"channel"`
	// Alert is kept raw to tell "absent" from "null": null asks for a data-only message.
	Alert json.RawMessage `json:"alert"`
	Extra map[string]any  `json:"extra"`
	FCM   struct {
		Data map[string]any `json:"data"`
	} `json:"fcm"`
}

// notificationFrom works out the visible text and the data of a push.
//
//   - Moodle sends its message details in "extra" and no alert: the title is the site's name and the
//     body is the short message.
//   - An alert that is a string is both title and body; an object may give its own title and body.
//   - An alert that is null, or no text at all, makes the message data only, which Moodle uses for
//     encrypted notifications.
//
// The whole of "extra" goes to the app as data, then "fcm.data" on top.
func notificationFrom(req pushRequest) (title, body string, data map[string]string) {
	data = stringify(req.Extra)
	for k, v := range stringify(req.FCM.Data) {
		data[k] = v
	}

	switch {
	case len(req.Alert) == 0:
		if text(req.Extra["processor"]) == "moodle" {
			title = text(req.Extra["sitefullname"])
			if title == "" {
				title = "Notification"
			}
			body = text(req.Extra["smallmessage"])
			if body == "" {
				body = text(req.Extra["fullmessage"])
			}
		}
	case string(req.Alert) == "null":
		// Data only.
	default:
		var plain string
		if json.Unmarshal(req.Alert, &plain) == nil {
			title, body = plain, plain
			break
		}
		var object map[string]any
		if json.Unmarshal(req.Alert, &object) == nil {
			title, body = text(object["title"]), text(object["body"])
		}
	}
	return title, body, data
}

// stringify turns values into the strings Firebase wants in its data: text as it is, booleans as 1 or 0,
// nested values as JSON, and nothing for null.
func stringify(values map[string]any) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		switch v := value.(type) {
		case nil:
		case string:
			out[key] = v
		case bool:
			if v {
				out[key] = "1"
			} else {
				out[key] = "0"
			}
		case json.Number:
			out[key] = v.String()
		case float64:
			out[key] = fmt.Sprint(v)
		default:
			if encoded, err := json.Marshal(v); err == nil {
				out[key] = string(encoded)
			}
		}
	}
	return out
}

func text(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
