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

type response struct {
	ResultCode string `xml:"header>resultCode"`
	ResultMsg  string `xml:"header>resultMsg"`
	Items      []item `xml:"body>items>item"`
	TotalCount string `xml:"body>totalCount"`
}

type gatewayError struct {
	ErrMsg     string `xml:"cmmMsgHeader>errMsg"`
	AuthMsg    string `xml:"cmmMsgHeader>returnAuthMsg"`
	ReasonCode string `xml:"cmmMsgHeader>returnReasonCode"`
}

func Parse(body []byte) ([]Trade, int, error) {
	root, err := rootName(body)
	if err != nil {
		return nil, 0, err
	}

	switch root {
	case "response":
		return parseResponse(body)
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

func parseResponse(body []byte) ([]Trade, int, error) {
	var r response
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
	trades := make([]Trade, 0, len(r.Items))
	for i, it := range r.Items {
		t, err := it.trade()
		if err != nil {
			return nil, 0, fmt.Errorf("molit: row %d (%s): %w", i+1, clean(it.AptNm), err)
		}
		trades = append(trades, t)
	}
	return trades, total, nil
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
