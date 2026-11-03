package molit

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type APIError struct {
	Code string
	Msg  string
}

func (e *APIError) Error() string { return fmt.Sprintf("molit api error %s: %s", e.Code, e.Msg) }

type response[I any] struct {
	ResultCode string `xml:"header>resultCode"`
	ResultMsg  string `xml:"header>resultMsg"`
	Items      []I    `xml:"body>items>item"`
	TotalCount string `xml:"body>totalCount"`
}

type apiItem interface {
	name() string
}

type gatewayError struct {
	ErrMsg     string `xml:"cmmMsgHeader>errMsg"`
	AuthMsg    string `xml:"cmmMsgHeader>returnAuthMsg"`
	ReasonCode string `xml:"cmmMsgHeader>returnReasonCode"`
}

func Parse(body []byte) ([]Trade, int, error) { return parse(body, item.trade) }

func ParseRents(body []byte) ([]Rent, int, error) { return parse(body, rentItem.rent) }

func parse[I apiItem, T any](body []byte, convert func(I) (T, error)) ([]T, int, error) {
	root, err := rootName(body)
	if err != nil {
		return nil, 0, err
	}

	switch root {
	case "response":
		return parseResponse(body, convert)
	case "OpenAPI_ServiceResponse":
		var g gatewayError
		if err := xml.Unmarshal(body, &g); err != nil {
			return nil, 0, fmt.Errorf("molit: decode gateway error: %w", err)
		}
		return nil, 0, &APIError{Code: clean(g.ReasonCode), Msg: clean(g.ErrMsg + " " + g.AuthMsg)}
	default:
		return nil, 0, fmt.Errorf("molit: unexpected response <%s>: %.120q", root, body)
	}
}

func parseResponse[I apiItem, T any](body []byte, convert func(I) (T, error)) ([]T, int, error) {
	var r response[I]
	if err := xml.Unmarshal(body, &r); err != nil {
		return nil, 0, fmt.Errorf("molit: decode response: %w", err)
	}
	if code := clean(r.ResultCode); code != "000" {
		return nil, 0, &APIError{Code: code, Msg: clean(r.ResultMsg)}
	}
	total, err := strconv.Atoi(clean(r.TotalCount))
	if err != nil {
		return nil, 0, fmt.Errorf("molit: bad totalCount %q", r.TotalCount)
	}
	out := make([]T, 0, len(r.Items))
	for i, it := range r.Items {
		t, err := convert(it)
		if err != nil {
			return nil, 0, fmt.Errorf("molit: row %d (%s): %w", i+1, it.name(), err)
		}
		out = append(out, t)
	}
	return out, total, nil
}

func rootName(body []byte) (string, error) {
	d := xml.NewDecoder(bytes.NewReader(body))
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("molit: no XML element in response: %.120q", body)
		}
		if err != nil {
			return "", fmt.Errorf("molit: malformed response: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, nil
		}
	}
}
