package vm

import (
	"encoding/base64"
	"fmt"
	"mime"
	"strings"
	"sync"
	"time"

	"github.com/tamnd/goipy/object"
)

// ── shared MIME class state ───────────────────────────────────────────────────

var (
	emailMimeBaseClsOnce sync.Once
	emailMimeBaseCls     *object.Class
	emailMimeNMPClsOnce  sync.Once
	emailMimeNMPCls      *object.Class
)

func (i *Interp) getMIMEBaseCls() *object.Class {
	emailMimeBaseClsOnce.Do(func() {
		emailMimeBaseCls = i.buildMIMEBaseClass()
	})
	return emailMimeBaseCls
}

func (i *Interp) getMIMENonMultipartCls() *object.Class {
	emailMimeNMPClsOnce.Do(func() {
		emailMimeNMPCls = i.buildMIMENonMultipartClass()
	})
	return emailMimeNMPCls
}

// ── MIMEBase ──────────────────────────────────────────────────────────────────

func (i *Interp) buildMIMEBaseClass() *object.Class {
	msgCls := i.getEmailMessageClass()
	cls := &object.Class{
		Name:  "MIMEBase",
		Bases: []*object.Class{msgCls},
		Dict:  object.NewDict(),
	}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "MIMEBase requires _maintype and _subtype")
			}
			self := a[0].(*object.Instance)
			// Call Message.__init__
			self.Dict.SetStr("_headers", &object.List{})
			self.Dict.SetStr("_payload", object.None)
			self.Dict.SetStr("_deftype", &object.Str{V: "text/plain"})
			self.Dict.SetStr("preamble", object.None)
			self.Dict.SetStr("epilogue", object.None)
			self.Dict.SetStr("defects", &object.List{})

			maintype := object.Str_(a[1])
			subtype := object.Str_(a[2])
			ct := maintype + "/" + subtype
			// Build params from kw
			params := map[string]string{}
			if kw != nil {
				dictIterStr(kw, func(k string, v object.Object) {
					if k == "policy" {
						return
					}
					k2 := strings.ReplaceAll(k, "_", "-")
					params[k2] = object.Str_(v)
				})
			}
			formatted := mime.FormatMediaType(ct, params)
			if formatted == "" {
				formatted = ct
			}
			emailHeaderSet(self, "MIME-Version", "1.0")
			emailHeaderSet(self, "Content-Type", formatted)
			return object.None, nil
		}})
	cls.Dict.SetStr("add_payload", &object.BuiltinFunc{Name: "add_payload",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			pl, ok := self.Dict.GetStr("_payload")
			var lst *object.List
			if ok {
				lst, _ = pl.(*object.List)
			}
			if lst == nil {
				lst = &object.List{}
				self.Dict.SetStr("_payload", lst)
			}
			lst.V = append(lst.V, a[1])
			return object.None, nil
		}})
	return cls
}

// ── MIMENonMultipart ──────────────────────────────────────────────────────────

func (i *Interp) buildMIMENonMultipartClass() *object.Class {
	baseCls := i.getMIMEBaseCls()
	cls := &object.Class{
		Name:  "MIMENonMultipart",
		Bases: []*object.Class{baseCls},
		Dict:  object.NewDict(),
	}
	cls.Dict.SetStr("attach", &object.BuiltinFunc{Name: "attach",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.typeErr, "MIMENonMultipart.attach() not allowed")
		}})
	return cls
}

// ── MIMEMultipart ─────────────────────────────────────────────────────────────

func (i *Interp) buildMIMEMultipartClass() *object.Class {
	baseCls := i.getMIMEBaseCls()
	cls := &object.Class{
		Name:  "MIMEMultipart",
		Bases: []*object.Class{baseCls},
		Dict:  object.NewDict(),
	}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			subtype := "mixed"
			if len(a) >= 2 && a[1] != object.None {
				if s, ok := a[1].(*object.Str); ok {
					subtype = s.V
				}
			}
			boundary := fmt.Sprintf("===============%d==", time.Now().UnixNano()%1e12)
			if len(a) >= 3 && a[2] != object.None {
				if s, ok := a[2].(*object.Str); ok {
					boundary = s.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("boundary"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						boundary = s.V
					}
				}
			}
			// Init base state
			self.Dict.SetStr("_headers", &object.List{})
			self.Dict.SetStr("_payload", &object.List{})
			self.Dict.SetStr("_deftype", &object.Str{V: "text/plain"})
			self.Dict.SetStr("preamble", object.None)
			self.Dict.SetStr("epilogue", object.None)
			self.Dict.SetStr("defects", &object.List{})

			params := map[string]string{"boundary": boundary}
			// Add any extra kw params (excluding boundary/policy)
			if kw != nil {
				dictIterStr(kw, func(k string, v object.Object) {
					if k == "boundary" || k == "policy" {
						return
					}
					params[strings.ReplaceAll(k, "_", "-")] = object.Str_(v)
				})
			}
			// Subparts
			if len(a) >= 4 && a[3] != object.None {
				if lst, ok := a[3].(*object.List); ok {
					self.Dict.SetStr("_payload", lst)
				}
			}
			ct := mime.FormatMediaType("multipart/"+subtype, params)
			if ct == "" {
				ct = fmt.Sprintf("multipart/%s; boundary=%q", subtype, boundary)
			}
			emailHeaderSet(self, "MIME-Version", "1.0")
			emailHeaderSet(self, "Content-Type", ct)
			return object.None, nil
		}})
	return cls
}

// ── MIMEText ──────────────────────────────────────────────────────────────────

func (i *Interp) buildMIMETextClass() *object.Class {
	nmpCls := i.getMIMENonMultipartCls()
	cls := &object.Class{
		Name:  "MIMEText",
		Bases: []*object.Class{nmpCls},
		Dict:  object.NewDict(),
	}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "MIMEText requires _text")
			}
			self := a[0].(*object.Instance)
			text := object.Str_(a[1])
			subtype := "plain"
			if len(a) >= 3 && a[2] != object.None {
				if s, ok := a[2].(*object.Str); ok {
					subtype = s.V
				}
			}
			charset := "us-ascii"
			if len(a) >= 4 && a[3] != object.None {
				charset = object.Str_(a[3])
			}
			if kw != nil {
				if v, ok := kw.GetStr("_subtype"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						subtype = s.V
					}
				}
				if v, ok := kw.GetStr("_charset"); ok && v != object.None {
					charset = object.Str_(v)
				}
			}
			// Auto-detect charset if nil/None
			if charset == "" || charset == "None" {
				isASCII := true
				for _, r := range text {
					if r > 127 {
						isASCII = false
						break
					}
				}
				if isASCII {
					charset = "us-ascii"
				} else {
					charset = "utf-8"
				}
			}

			self.Dict.SetStr("_headers", &object.List{})
			self.Dict.SetStr("_deftype", &object.Str{V: "text/plain"})
			self.Dict.SetStr("preamble", object.None)
			self.Dict.SetStr("epilogue", object.None)
			self.Dict.SetStr("defects", &object.List{})

			params := map[string]string{"charset": charset}
			ct := mime.FormatMediaType("text/"+subtype, params)
			if ct == "" {
				ct = fmt.Sprintf("text/%s; charset=%s", subtype, charset)
			}
			emailHeaderSet(self, "MIME-Version", "1.0")
			emailHeaderSet(self, "Content-Type", ct)
			self.Dict.SetStr("_payload", &object.Str{V: text})
			return object.None, nil
		}})
	return cls
}

// ── MIMEApplication ───────────────────────────────────────────────────────────

func (i *Interp) buildMIMEApplicationClass() *object.Class {
	nmpCls := i.getMIMENonMultipartCls()
	cls := &object.Class{
		Name:  "MIMEApplication",
		Bases: []*object.Class{nmpCls},
		Dict:  object.NewDict(),
	}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "MIMEApplication requires _data")
			}
			self := a[0].(*object.Instance)
			var data []byte
			switch v := a[1].(type) {
			case *object.Bytes:
				data = v.V
			case *object.Str:
				data = []byte(v.V)
			default:
				data = []byte(object.Str_(a[1]))
			}
			subtype := "octet-stream"
			if len(a) >= 3 && a[2] != object.None {
				if s, ok := a[2].(*object.Str); ok {
					subtype = s.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("_subtype"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						subtype = s.V
					}
				}
			}

			self.Dict.SetStr("_headers", &object.List{})
			self.Dict.SetStr("_deftype", &object.Str{V: "text/plain"})
			self.Dict.SetStr("preamble", object.None)
			self.Dict.SetStr("epilogue", object.None)
			self.Dict.SetStr("defects", &object.List{})

			params := map[string]string{}
			if kw != nil {
				dictIterStr(kw, func(k string, v object.Object) {
					if k == "_subtype" || k == "_encoder" || k == "policy" {
						return
					}
					params[strings.ReplaceAll(k, "_", "-")] = object.Str_(v)
				})
			}
			ct := mime.FormatMediaType("application/"+subtype, params)
			if ct == "" {
				ct = "application/" + subtype
			}
			emailHeaderSet(self, "MIME-Version", "1.0")
			emailHeaderSet(self, "Content-Type", ct)

			// Apply encoder if provided (default: encode_base64)
			var encoderFn object.Object
			if len(a) >= 4 {
				encoderFn = a[3]
			} else if kw != nil {
				if v, ok := kw.GetStr("_encoder"); ok {
					encoderFn = v
				}
			}
			if encoderFn != nil && encoderFn != object.None {
				// Store raw bytes first
				self.Dict.SetStr("_payload", &object.Bytes{V: data})
			} else {
				// Default: base64 encode
				encoded := base64.StdEncoding.EncodeToString(data)
				self.Dict.SetStr("_payload", &object.Str{V: encoded})
				emailHeaderSet(self, "Content-Transfer-Encoding", "base64")
			}
			return object.None, nil
		}})
	return cls
}

// ── MIMEImage ─────────────────────────────────────────────────────────────────

func (i *Interp) buildMIMEImageClass() *object.Class {
	nmpCls := i.getMIMENonMultipartCls()
	cls := &object.Class{
		Name:  "MIMEImage",
		Bases: []*object.Class{nmpCls},
		Dict:  object.NewDict(),
	}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "MIMEImage requires _imagedata")
			}
			self := a[0].(*object.Instance)
			var data []byte
			switch v := a[1].(type) {
			case *object.Bytes:
				data = v.V
			case *object.Str:
				data = []byte(v.V)
			}
			subtype := "jpeg"
			if len(a) >= 3 && a[2] != object.None {
				if s, ok := a[2].(*object.Str); ok {
					subtype = s.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("_subtype"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						subtype = s.V
					}
				}
			}
			self.Dict.SetStr("_headers", &object.List{})
			self.Dict.SetStr("_deftype", &object.Str{V: "text/plain"})
			self.Dict.SetStr("preamble", object.None)
			self.Dict.SetStr("epilogue", object.None)
			self.Dict.SetStr("defects", &object.List{})
			emailHeaderSet(self, "MIME-Version", "1.0")
			emailHeaderSet(self, "Content-Type", "image/"+subtype)
			encoded := base64.StdEncoding.EncodeToString(data)
			self.Dict.SetStr("_payload", &object.Str{V: encoded})
			emailHeaderSet(self, "Content-Transfer-Encoding", "base64")
			return object.None, nil
		}})
	return cls
}

// ── MIMEAudio ─────────────────────────────────────────────────────────────────

func (i *Interp) buildMIMEAudioClass() *object.Class {
	nmpCls := i.getMIMENonMultipartCls()
	cls := &object.Class{
		Name:  "MIMEAudio",
		Bases: []*object.Class{nmpCls},
		Dict:  object.NewDict(),
	}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "MIMEAudio requires _audiodata")
			}
			self := a[0].(*object.Instance)
			var data []byte
			switch v := a[1].(type) {
			case *object.Bytes:
				data = v.V
			case *object.Str:
				data = []byte(v.V)
			}
			subtype := "wav"
			if len(a) >= 3 && a[2] != object.None {
				if s, ok := a[2].(*object.Str); ok {
					subtype = s.V
				}
			}
			self.Dict.SetStr("_headers", &object.List{})
			self.Dict.SetStr("_deftype", &object.Str{V: "text/plain"})
			self.Dict.SetStr("preamble", object.None)
			self.Dict.SetStr("epilogue", object.None)
			self.Dict.SetStr("defects", &object.List{})
			emailHeaderSet(self, "MIME-Version", "1.0")
			emailHeaderSet(self, "Content-Type", "audio/"+subtype)
			encoded := base64.StdEncoding.EncodeToString(data)
			self.Dict.SetStr("_payload", &object.Str{V: encoded})
			emailHeaderSet(self, "Content-Transfer-Encoding", "base64")
			return object.None, nil
		}})
	return cls
}

// ── MIMEMessage ───────────────────────────────────────────────────────────────

func (i *Interp) buildMIMEMessageClass() *object.Class {
	nmpCls := i.getMIMENonMultipartCls()
	cls := &object.Class{
		Name:  "MIMEMessage",
		Bases: []*object.Class{nmpCls},
		Dict:  object.NewDict(),
	}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "MIMEMessage requires _msg")
			}
			self := a[0].(*object.Instance)
			subtype := "rfc822"
			if len(a) >= 3 && a[2] != object.None {
				if s, ok := a[2].(*object.Str); ok {
					subtype = s.V
				}
			}
			self.Dict.SetStr("_headers", &object.List{})
			self.Dict.SetStr("_deftype", &object.Str{V: "text/plain"})
			self.Dict.SetStr("preamble", object.None)
			self.Dict.SetStr("epilogue", object.None)
			self.Dict.SetStr("defects", &object.List{})
			emailHeaderSet(self, "MIME-Version", "1.0")
			emailHeaderSet(self, "Content-Type", "message/"+subtype)
			self.Dict.SetStr("_payload", a[1])
			return object.None, nil
		}})
	return cls
}

// ── email.mime submodule builders ─────────────────────────────────────────────

func (i *Interp) buildEmailMime() *object.Module {
	m := &object.Module{Name: "email.mime", Dict: object.NewDict()}
	return m
}

func (i *Interp) buildEmailMimeBase() *object.Module {
	m := &object.Module{Name: "email.mime.base", Dict: object.NewDict()}
	m.Dict.SetStr("MIMEBase", i.getMIMEBaseCls())
	return m
}

func (i *Interp) buildEmailMimeNonMultipart() *object.Module {
	m := &object.Module{Name: "email.mime.nonmultipart", Dict: object.NewDict()}
	m.Dict.SetStr("MIMENonMultipart", i.getMIMENonMultipartCls())
	return m
}

func (i *Interp) buildEmailMimeMultipart() *object.Module {
	m := &object.Module{Name: "email.mime.multipart", Dict: object.NewDict()}
	m.Dict.SetStr("MIMEMultipart", i.buildMIMEMultipartClass())
	return m
}

func (i *Interp) buildEmailMimeText() *object.Module {
	m := &object.Module{Name: "email.mime.text", Dict: object.NewDict()}
	m.Dict.SetStr("MIMEText", i.buildMIMETextClass())
	return m
}

func (i *Interp) buildEmailMimeApplication() *object.Module {
	m := &object.Module{Name: "email.mime.application", Dict: object.NewDict()}
	m.Dict.SetStr("MIMEApplication", i.buildMIMEApplicationClass())
	return m
}

func (i *Interp) buildEmailMimeImage() *object.Module {
	m := &object.Module{Name: "email.mime.image", Dict: object.NewDict()}
	m.Dict.SetStr("MIMEImage", i.buildMIMEImageClass())
	return m
}

func (i *Interp) buildEmailMimeAudio() *object.Module {
	m := &object.Module{Name: "email.mime.audio", Dict: object.NewDict()}
	m.Dict.SetStr("MIMEAudio", i.buildMIMEAudioClass())
	return m
}

func (i *Interp) buildEmailMimeMessage() *object.Module {
	m := &object.Module{Name: "email.mime.message", Dict: object.NewDict()}
	m.Dict.SetStr("MIMEMessage", i.buildMIMEMessageClass())
	return m
}
