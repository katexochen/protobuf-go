// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package impl

import (
	"sort"

	"google.golang.org/protobuf/internal/encoding/messageset"
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/flags"
)

func sizeMessageSet(mi *MessageInfo, p pointer, opts marshalOptions) (size int) {
	if !flags.ProtoLegacy {
		return 0
	}

	ext := *p.Apply(mi.extensionOffset).Extensions()
	for _, x := range ext {
		xi := getExtensionFieldInfo(x.Type())
		if xi.funcs.size == nil {
			continue
		}
		num, _ := wire.DecodeTag(xi.wiretag)
		size += messageset.SizeField(num)
		size += xi.funcs.size(x.Value(), wire.SizeTag(messageset.FieldMessage), opts)
	}

	unknown := *p.Apply(mi.unknownOffset).Bytes()
	size += messageset.SizeUnknown(unknown)

	return size
}

func marshalMessageSet(mi *MessageInfo, b []byte, p pointer, opts marshalOptions) ([]byte, error) {
	if !flags.ProtoLegacy {
		return b, errors.New("no support for message_set_wire_format")
	}

	ext := *p.Apply(mi.extensionOffset).Extensions()
	switch len(ext) {
	case 0:
	case 1:
		// Fast-path for one extension: Don't bother sorting the keys.
		for _, x := range ext {
			var err error
			b, err = marshalMessageSetField(mi, b, x, opts)
			if err != nil {
				return b, err
			}
		}
	default:
		// Sort the keys to provide a deterministic encoding.
		// Not sure this is required, but the old code does it.
		keys := make([]int, 0, len(ext))
		for k := range ext {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		for _, k := range keys {
			var err error
			b, err = marshalMessageSetField(mi, b, ext[int32(k)], opts)
			if err != nil {
				return b, err
			}
		}
	}

	unknown := *p.Apply(mi.unknownOffset).Bytes()
	b, err := messageset.AppendUnknown(b, unknown)
	if err != nil {
		return b, err
	}

	return b, nil
}

func marshalMessageSetField(mi *MessageInfo, b []byte, x ExtensionField, opts marshalOptions) ([]byte, error) {
	xi := getExtensionFieldInfo(x.Type())
	num, _ := wire.DecodeTag(xi.wiretag)
	b = messageset.AppendFieldStart(b, num)
	b, err := xi.funcs.marshal(b, x.Value(), wire.EncodeTag(messageset.FieldMessage, wire.BytesType), opts)
	if err != nil {
		return b, err
	}
	b = messageset.AppendFieldEnd(b)
	return b, nil
}

func unmarshalMessageSet(mi *MessageInfo, b []byte, p pointer, opts unmarshalOptions) (int, error) {
	if !flags.ProtoLegacy {
		return 0, errors.New("no support for message_set_wire_format")
	}

	ep := p.Apply(mi.extensionOffset).Extensions()
	if *ep == nil {
		*ep = make(map[int32]ExtensionField)
	}
	ext := *ep
	unknown := p.Apply(mi.unknownOffset).Bytes()
	err := messageset.Unmarshal(b, true, func(num wire.Number, v []byte) error {
		_, err := mi.unmarshalExtension(v, num, wire.BytesType, ext, opts)
		if err == errUnknown {
			*unknown = wire.AppendTag(*unknown, num, wire.BytesType)
			*unknown = append(*unknown, v...)
			return nil
		}
		return err
	})
	return len(b), err
}
