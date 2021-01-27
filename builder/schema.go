package builder

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Schema extends Terraform Schema with metadata
type Schema struct {
	Name     string
	FullName string

	schema.Schema
}

type schemaBuilder struct {
	field  protoreflect.FieldDescriptor
	schema *Schema
}

func (b *schemaBuilder) setName() {
	b.schema.Name = string(b.field.Name())
}

func (b *schemaBuilder) setFullName() {
	b.schema.FullName = string(b.field.FullName())
}

func (b *schemaBuilder) setRequired() {
	b.schema.Required = b.field.Cardinality() == protoreflect.Required
}

// Returns true if current field contains nested resource (and not the list of nested resources)
func (b *schemaBuilder) isNestedResource() bool {
	return b.field.Kind() == protoreflect.MessageKind && !b.field.IsList()
}

func (b *schemaBuilder) setTypeAndElem() {
	kind := b.field.Kind()

	if b.field.IsMap() {
		b.schema.Type = schema.TypeMap
		b.setElem(b.field.MapValue().Kind(), b.field.MapValue().Message())
	} else if b.field.IsList() {
		b.schema.Type = schema.TypeList
		b.setElem(b.field.Kind(), b.field.Message())
	} else if b.isNestedResource() {
		// If the nested resource is another structure, than we should produce a list with the single item
		// That's the weirdo way Terraform handles such case
		b.schema.Type = schema.TypeList
		b.schema.MaxItems = 1
		b.setElem(b.field.Kind(), b.field.Message())
	} else {
		b.schema.Type = b.getTypeFromKind(kind)
	}
}

func (b *schemaBuilder) setElem(kind protoreflect.Kind, message protoreflect.MessageDescriptor) {
	var elem interface{}

	if kind == protoreflect.MessageKind {
		elem = BuildResourceFromMessage(&message)
	} else {
		s := Schema{}
		s.Type = b.getTypeFromKind(kind)
		elem = s
	}

	b.schema.Elem = elem
}

func (b *schemaBuilder) getTypeFromKind(kind protoreflect.Kind) schema.ValueType {
	switch kind {
	case protoreflect.BoolKind:
		return schema.TypeBool
	case protoreflect.StringKind, protoreflect.BytesKind, protoreflect.EnumKind:
		return schema.TypeString
	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Uint64Kind:
		return schema.TypeInt
	case protoreflect.DoubleKind:
		return schema.TypeFloat
	}

	// TODO: proper error handling here
	log.Fatalf("Unknown schema kind %s!", kind.GoString())

	return -1
}

// BuildSchemaFromField builds resource from protoreflect message
func BuildSchemaFromField(field *protoreflect.FieldDescriptor) *Schema {
	schema := &Schema{}

	b := schemaBuilder{field: *field, schema: schema}

	b.setName()
	b.setFullName()
	b.setRequired()
	b.setTypeAndElem()

	return schema
}