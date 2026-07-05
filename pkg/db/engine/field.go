package engine

import (
	"github.com/amadigan/flit/pkg/db"
	"github.com/amadigan/flit/pkg/schema"
)

func parseText(index int, header db.ValueHeader, cursor db.ValueCursor) ([]Text, error) {
	switch header.Type() {
	case db.TypeString:
		str, err := parseString(index, header, cursor)
		if err != nil {
			return nil, err
		}
		return []Text{{Content: []string{str}}}, nil
	case db.TypeObject:
		text, err := parseTextObject(index, header, cursor)
		if err != nil {
			return nil, err
		}
		return []Text{text}, nil
	case db.TypeArray:
		subheader, err := cursor.ObjectHeader(index)
		if err != nil {
			return nil, err
		}

		subcursor, err := cursor.Object(index)
		if err != nil {
			return nil, err
		}

		texts := make([]Text, len(subheader))
		for i := 0; i < len(subheader); i++ {
			switch subheader[i].Type() {
			case db.TypeString:
				scalar, err := subcursor.Scalar(i)
				if err != nil {
					return nil, err
				}
				str, err := scalar.AsString()
				if err != nil {
					return nil, err
				}
				texts[i] = Text{Content: []string{str}}
			case db.TypeObject:
				text, err := parseTextObject(i, subheader[i], subcursor)
				if err != nil {
					return nil, err
				}
				texts[i] = text
			default:
				return nil, db.ErrMalformedData
			}
		}
	default:
		return nil, db.ErrMalformedData
	}

	return nil, nil
}

func parseString(index int, header db.ValueHeader, cursor db.ValueCursor) (string, error) {
	scalar, err := cursor.Scalar(index)
	if err != nil {
		return "", err
	}
	str, err := scalar.AsString()
	if err != nil {
		return "", err
	}
	return str, nil
}

func (e *LiveEngine) parseField(index int, headers []db.ValueHeader, cursor db.ValueCursor) (Field, error) {
	var field Field
	var err error

	switch headers[index].Type() {
	case db.TypeObject:
		field, err = e.parseFieldObject(index, cursor)
	case db.TypeArray:
		field.Content, err = parseTextArray(index, cursor)
	case db.TypeString:
		field.Content = []Text{{Content: []string{""}}}
		field.Content[0].Content[0], err = parseString(index, headers[index], cursor)
	default:
		return Field{}, db.ErrMalformedData
	}

	if err != nil {
		return Field{}, err
	}

	return field, nil
}

func (e *LiveEngine) parseFieldObject(index int, cursor db.ValueCursor) (Field, error) {
	headers, err := cursor.ObjectHeader(index)
	if err != nil {
		return Field{}, err
	}

	cursor, err = cursor.Object(index)
	if err != nil {
		return Field{}, err
	}

	var sourceIdStr string
	var sourceId schema.DocumentId
	var line, column, endLine, endColumn, offset, length int
	var content string
	var contentList []Text
	var code string

	for i := 0; i < len(headers); i++ {
		name, err := cursor.Name(i)
		if err != nil {
			return Field{}, err
		}
		switch name {
		case "sourceId":
			switch headers[i].Type() {
			case db.TypeString:
				sourceIdStr, err = parseString(i, headers[i], cursor)
			case db.TypeInt:
				scalar, err := cursor.Scalar(i)
				if err != nil {
					return Field{}, err
				}
				id, err := scalar.AsInt32()
				sourceId = schema.DocumentId(id)
			default:
				return Field{}, db.ErrMalformedData
			}
		case "line":
			line, err = parseIntField(i, cursor)
		case "column":
			column, err = parseIntField(i, cursor)
		case "endLine":
			endLine, err = parseIntField(i, cursor)
		case "endColumn":
			endColumn, err = parseIntField(i, cursor)
		case "offset":
			offset, err = parseIntField(i, cursor)
		case "length":
			length, err = parseIntField(i, cursor)
		case "content":
			switch headers[i].Type() {
			case db.TypeString:
				content, err = parseString(i, headers[i], cursor)
			case db.TypeArray:
				contentList, err = parseTextArray(i, cursor)
			default:
				return Field{}, db.ErrMalformedData
			}
		case "code":
			code, err = parseString(i, headers[i], cursor)
		}
		if err != nil {
			return Field{}, err
		}
	}

	if sourceIdStr != "" {
		sourceId, _ = e.idtable.Get(sourceIdStr)
	}

	field := Field{Code: code}
	if sourceId != 0 {
		field.Location = &schema.Ref{
			SourceId:  sourceId,
			Line:      line,
			Column:    column,
			EndLine:   endLine,
			EndColumn: endColumn,
			Offset:    offset,
			Length:    length,
		}
	}

	if len(contentList) > 0 {
		field.Content = contentList
	} else if content != "" {
		field.Content = []Text{{Content: []string{content}}}
	}

	return field, nil
}

func parseIntField(index int, cursor db.ValueCursor) (int, error) {
	scalar, err := cursor.Scalar(index)
	if err != nil {
		return 0, err
	}
	val, err := scalar.AsInt64()
	if err != nil {
		return 0, err
	}
	return int(val), nil
}

func parseTextArray(index int, cursor db.ValueCursor) ([]Text, error) {
	subheader, err := cursor.ObjectHeader(index)
	if err != nil {
		return nil, err
	}

	subcursor, err := cursor.Object(index)
	if err != nil {
		return nil, err
	}

	texts := make([]Text, len(subheader))
	for i := 0; i < len(subheader); i++ {
		switch subheader[i].Type() {
		case db.TypeString:
			str, err := parseString(i, subheader[i], subcursor)
			if err != nil {
				return nil, err
			}
			texts[i] = Text{Content: []string{str}}
		case db.TypeObject:
			text, err := parseTextObject(i, subheader[i], subcursor)
			if err != nil {
				return nil, err
			}
			texts[i] = text
		default:
			return nil, db.ErrMalformedData
		}
	}

	return texts, nil
}

func parseStringArray(index int, cursor db.ValueCursor) ([]string, error) {
	subheader, err := cursor.ObjectHeader(index)
	if err != nil {
		return nil, err
	}

	subcursor, err := cursor.Object(index)
	if err != nil {
		return nil, err
	}

	strings := make([]string, len(subheader))
	for i := 0; i < len(subheader); i++ {
		scalar, err := subcursor.Scalar(i)
		if err != nil {
			return nil, err
		}
		strings[i], err = scalar.AsString()
		if err != nil {
			return nil, err
		}
	}

	return strings, nil
}

func parseTextObject(index int, header db.ValueHeader, cursor db.ValueCursor) (Text, error) {
	subheader, err := cursor.ObjectHeader(index)
	if err != nil {
		return Text{}, err
	}

	subcursor, err := cursor.Object(index)
	if err != nil {
		return Text{}, err
	}

	var text Text

	for i := 0; i < len(subheader); i++ {
		name, err := subcursor.Name(i)
		if err != nil {
			return Text{}, err
		}
		switch name {
		case "content":
			switch subheader[i].Type() {
			case db.TypeString:
				str, err := parseString(i, subheader[i], subcursor)
				if err != nil {
					return Text{}, err
				}
				text.Content = []string{str}
			case db.TypeArray:
				contentList, err := parseStringArray(i, subcursor)
				if err != nil {
					return Text{}, err
				}
				text.Content = contentList
			default:
				return Text{}, db.ErrMalformedData
			}
		case "code":
			code, err := parseString(i, subheader[i], subcursor)
			if err != nil {
				return Text{}, err
			}
			text.Code = code
		default:
			return Text{}, db.ErrMalformedData
		}
	}

	return text, nil
}
