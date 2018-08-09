// This file is generated. Do not edit
// @generated

// https://github.com/Manishearth/rust-clippy/issues/702
#![allow(unknown_lints)]
#![allow(clippy)]

#![cfg_attr(rustfmt, rustfmt_skip)]

#![allow(box_pointers)]
#![allow(dead_code)]
#![allow(missing_docs)]
#![allow(non_camel_case_types)]
#![allow(non_snake_case)]
#![allow(non_upper_case_globals)]
#![allow(trivial_casts)]
#![allow(unsafe_code)]
#![allow(unused_imports)]
#![allow(unused_results)]

use protobuf::Message as Message_imported_for_functions;
use protobuf::ProtobufEnum as ProtobufEnum_imported_for_functions;

#[derive(PartialEq,Clone,Default)]
pub struct OpenRequest {
    // message fields
    pub uuid: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for OpenRequest {}

impl OpenRequest {
    pub fn new() -> OpenRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static OpenRequest {
        static mut instance: ::protobuf::lazy::Lazy<OpenRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const OpenRequest,
        };
        unsafe {
            instance.get(OpenRequest::new)
        }
    }

    // bytes uuid = 1;

    pub fn clear_uuid(&mut self) {
        self.uuid.clear();
    }

    // Param is passed by value, moved
    pub fn set_uuid(&mut self, v: ::std::vec::Vec<u8>) {
        self.uuid = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_uuid(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.uuid
    }

    // Take field
    pub fn take_uuid(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.uuid, ::std::vec::Vec::new())
    }

    pub fn get_uuid(&self) -> &[u8] {
        &self.uuid
    }

    fn get_uuid_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.uuid
    }

    fn mut_uuid_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.uuid
    }
}

impl ::protobuf::Message for OpenRequest {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.uuid)?;
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if !self.uuid.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.uuid);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.uuid.is_empty() {
            os.write_bytes(1, &self.uuid)?;
        }
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for OpenRequest {
    fn new() -> OpenRequest {
        OpenRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<OpenRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "uuid",
                    OpenRequest::get_uuid_for_reflect,
                    OpenRequest::mut_uuid_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<OpenRequest>(
                    "OpenRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for OpenRequest {
    fn clear(&mut self) {
        self.clear_uuid();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for OpenRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for OpenRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct OpenResponse {
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for OpenResponse {}

impl OpenResponse {
    pub fn new() -> OpenResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static OpenResponse {
        static mut instance: ::protobuf::lazy::Lazy<OpenResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const OpenResponse,
        };
        unsafe {
            instance.get(OpenResponse::new)
        }
    }
}

impl ::protobuf::Message for OpenResponse {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for OpenResponse {
    fn new() -> OpenResponse {
        OpenResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<OpenResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let fields = ::std::vec::Vec::new();
                ::protobuf::reflect::MessageDescriptor::new::<OpenResponse>(
                    "OpenResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for OpenResponse {
    fn clear(&mut self) {
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for OpenResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for OpenResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct WriteHead {
    // message fields
    pub uuid: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for WriteHead {}

impl WriteHead {
    pub fn new() -> WriteHead {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static WriteHead {
        static mut instance: ::protobuf::lazy::Lazy<WriteHead> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const WriteHead,
        };
        unsafe {
            instance.get(WriteHead::new)
        }
    }

    // bytes uuid = 1;

    pub fn clear_uuid(&mut self) {
        self.uuid.clear();
    }

    // Param is passed by value, moved
    pub fn set_uuid(&mut self, v: ::std::vec::Vec<u8>) {
        self.uuid = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_uuid(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.uuid
    }

    // Take field
    pub fn take_uuid(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.uuid, ::std::vec::Vec::new())
    }

    pub fn get_uuid(&self) -> &[u8] {
        &self.uuid
    }

    fn get_uuid_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.uuid
    }

    fn mut_uuid_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.uuid
    }
}

impl ::protobuf::Message for WriteHead {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.uuid)?;
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if !self.uuid.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.uuid);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.uuid.is_empty() {
            os.write_bytes(1, &self.uuid)?;
        }
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for WriteHead {
    fn new() -> WriteHead {
        WriteHead::new()
    }

    fn descriptor_static(_: ::std::option::Option<WriteHead>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "uuid",
                    WriteHead::get_uuid_for_reflect,
                    WriteHead::mut_uuid_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<WriteHead>(
                    "WriteHead",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for WriteHead {
    fn clear(&mut self) {
        self.clear_uuid();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for WriteHead {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for WriteHead {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct Mutation {
    // message fields
    pub op: Mutation_OP,
    pub key: ::std::vec::Vec<u8>,
    pub value: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for Mutation {}

impl Mutation {
    pub fn new() -> Mutation {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static Mutation {
        static mut instance: ::protobuf::lazy::Lazy<Mutation> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const Mutation,
        };
        unsafe {
            instance.get(Mutation::new)
        }
    }

    // .import_kvpb.Mutation.OP op = 1;

    pub fn clear_op(&mut self) {
        self.op = Mutation_OP::Put;
    }

    // Param is passed by value, moved
    pub fn set_op(&mut self, v: Mutation_OP) {
        self.op = v;
    }

    pub fn get_op(&self) -> Mutation_OP {
        self.op
    }

    fn get_op_for_reflect(&self) -> &Mutation_OP {
        &self.op
    }

    fn mut_op_for_reflect(&mut self) -> &mut Mutation_OP {
        &mut self.op
    }

    // bytes key = 2;

    pub fn clear_key(&mut self) {
        self.key.clear();
    }

    // Param is passed by value, moved
    pub fn set_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.key
    }

    // Take field
    pub fn take_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.key, ::std::vec::Vec::new())
    }

    pub fn get_key(&self) -> &[u8] {
        &self.key
    }

    fn get_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.key
    }

    fn mut_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.key
    }

    // bytes value = 3;

    pub fn clear_value(&mut self) {
        self.value.clear();
    }

    // Param is passed by value, moved
    pub fn set_value(&mut self, v: ::std::vec::Vec<u8>) {
        self.value = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_value(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.value
    }

    // Take field
    pub fn take_value(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.value, ::std::vec::Vec::new())
    }

    pub fn get_value(&self) -> &[u8] {
        &self.value
    }

    fn get_value_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.value
    }

    fn mut_value_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.value
    }
}

impl ::protobuf::Message for Mutation {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_enum()?;
                    self.op = tmp;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.value)?;
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if self.op != Mutation_OP::Put {
            my_size += ::protobuf::rt::enum_size(1, self.op);
        }
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.key);
        }
        if !self.value.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.value);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.op != Mutation_OP::Put {
            os.write_enum(1, self.op.value())?;
        }
        if !self.key.is_empty() {
            os.write_bytes(2, &self.key)?;
        }
        if !self.value.is_empty() {
            os.write_bytes(3, &self.value)?;
        }
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for Mutation {
    fn new() -> Mutation {
        Mutation::new()
    }

    fn descriptor_static(_: ::std::option::Option<Mutation>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeEnum<Mutation_OP>>(
                    "op",
                    Mutation::get_op_for_reflect,
                    Mutation::mut_op_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    Mutation::get_key_for_reflect,
                    Mutation::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "value",
                    Mutation::get_value_for_reflect,
                    Mutation::mut_value_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<Mutation>(
                    "Mutation",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for Mutation {
    fn clear(&mut self) {
        self.clear_op();
        self.clear_key();
        self.clear_value();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for Mutation {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for Mutation {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(Clone,PartialEq,Eq,Debug,Hash)]
pub enum Mutation_OP {
    Put = 0,
}

impl ::protobuf::ProtobufEnum for Mutation_OP {
    fn value(&self) -> i32 {
        *self as i32
    }

    fn from_i32(value: i32) -> ::std::option::Option<Mutation_OP> {
        match value {
            0 => ::std::option::Option::Some(Mutation_OP::Put),
            _ => ::std::option::Option::None
        }
    }

    fn values() -> &'static [Self] {
        static values: &'static [Mutation_OP] = &[
            Mutation_OP::Put,
        ];
        values
    }

    fn enum_descriptor_static(_: ::std::option::Option<Mutation_OP>) -> &'static ::protobuf::reflect::EnumDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::EnumDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::EnumDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                ::protobuf::reflect::EnumDescriptor::new("Mutation_OP", file_descriptor_proto())
            })
        }
    }
}

impl ::std::marker::Copy for Mutation_OP {
}

impl ::std::default::Default for Mutation_OP {
    fn default() -> Self {
        Mutation_OP::Put
    }
}

impl ::protobuf::reflect::ProtobufValue for Mutation_OP {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Enum(self.descriptor())
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct WriteBatch {
    // message fields
    pub commit_ts: u64,
    pub mutations: ::protobuf::RepeatedField<Mutation>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for WriteBatch {}

impl WriteBatch {
    pub fn new() -> WriteBatch {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static WriteBatch {
        static mut instance: ::protobuf::lazy::Lazy<WriteBatch> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const WriteBatch,
        };
        unsafe {
            instance.get(WriteBatch::new)
        }
    }

    // uint64 commit_ts = 1;

    pub fn clear_commit_ts(&mut self) {
        self.commit_ts = 0;
    }

    // Param is passed by value, moved
    pub fn set_commit_ts(&mut self, v: u64) {
        self.commit_ts = v;
    }

    pub fn get_commit_ts(&self) -> u64 {
        self.commit_ts
    }

    fn get_commit_ts_for_reflect(&self) -> &u64 {
        &self.commit_ts
    }

    fn mut_commit_ts_for_reflect(&mut self) -> &mut u64 {
        &mut self.commit_ts
    }

    // repeated .import_kvpb.Mutation mutations = 2;

    pub fn clear_mutations(&mut self) {
        self.mutations.clear();
    }

    // Param is passed by value, moved
    pub fn set_mutations(&mut self, v: ::protobuf::RepeatedField<Mutation>) {
        self.mutations = v;
    }

    // Mutable pointer to the field.
    pub fn mut_mutations(&mut self) -> &mut ::protobuf::RepeatedField<Mutation> {
        &mut self.mutations
    }

    // Take field
    pub fn take_mutations(&mut self) -> ::protobuf::RepeatedField<Mutation> {
        ::std::mem::replace(&mut self.mutations, ::protobuf::RepeatedField::new())
    }

    pub fn get_mutations(&self) -> &[Mutation] {
        &self.mutations
    }

    fn get_mutations_for_reflect(&self) -> &::protobuf::RepeatedField<Mutation> {
        &self.mutations
    }

    fn mut_mutations_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<Mutation> {
        &mut self.mutations
    }
}

impl ::protobuf::Message for WriteBatch {
    fn is_initialized(&self) -> bool {
        for v in &self.mutations {
            if !v.is_initialized() {
                return false;
            }
        };
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.commit_ts = tmp;
                },
                2 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.mutations)?;
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if self.commit_ts != 0 {
            my_size += ::protobuf::rt::value_size(1, self.commit_ts, ::protobuf::wire_format::WireTypeVarint);
        }
        for value in &self.mutations {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.commit_ts != 0 {
            os.write_uint64(1, self.commit_ts)?;
        }
        for v in &self.mutations {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        };
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for WriteBatch {
    fn new() -> WriteBatch {
        WriteBatch::new()
    }

    fn descriptor_static(_: ::std::option::Option<WriteBatch>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "commit_ts",
                    WriteBatch::get_commit_ts_for_reflect,
                    WriteBatch::mut_commit_ts_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Mutation>>(
                    "mutations",
                    WriteBatch::get_mutations_for_reflect,
                    WriteBatch::mut_mutations_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<WriteBatch>(
                    "WriteBatch",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for WriteBatch {
    fn clear(&mut self) {
        self.clear_commit_ts();
        self.clear_mutations();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for WriteBatch {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for WriteBatch {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct WriteRequest {
    // message oneof groups
    chunk: ::std::option::Option<WriteRequest_oneof_chunk>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for WriteRequest {}

#[derive(Clone,PartialEq)]
pub enum WriteRequest_oneof_chunk {
    head(WriteHead),
    batch(WriteBatch),
}

impl WriteRequest {
    pub fn new() -> WriteRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static WriteRequest {
        static mut instance: ::protobuf::lazy::Lazy<WriteRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const WriteRequest,
        };
        unsafe {
            instance.get(WriteRequest::new)
        }
    }

    // .import_kvpb.WriteHead head = 1;

    pub fn clear_head(&mut self) {
        self.chunk = ::std::option::Option::None;
    }

    pub fn has_head(&self) -> bool {
        match self.chunk {
            ::std::option::Option::Some(WriteRequest_oneof_chunk::head(..)) => true,
            _ => false,
        }
    }

    // Param is passed by value, moved
    pub fn set_head(&mut self, v: WriteHead) {
        self.chunk = ::std::option::Option::Some(WriteRequest_oneof_chunk::head(v))
    }

    // Mutable pointer to the field.
    pub fn mut_head(&mut self) -> &mut WriteHead {
        if let ::std::option::Option::Some(WriteRequest_oneof_chunk::head(_)) = self.chunk {
        } else {
            self.chunk = ::std::option::Option::Some(WriteRequest_oneof_chunk::head(WriteHead::new()));
        }
        match self.chunk {
            ::std::option::Option::Some(WriteRequest_oneof_chunk::head(ref mut v)) => v,
            _ => panic!(),
        }
    }

    // Take field
    pub fn take_head(&mut self) -> WriteHead {
        if self.has_head() {
            match self.chunk.take() {
                ::std::option::Option::Some(WriteRequest_oneof_chunk::head(v)) => v,
                _ => panic!(),
            }
        } else {
            WriteHead::new()
        }
    }

    pub fn get_head(&self) -> &WriteHead {
        match self.chunk {
            ::std::option::Option::Some(WriteRequest_oneof_chunk::head(ref v)) => v,
            _ => WriteHead::default_instance(),
        }
    }

    // .import_kvpb.WriteBatch batch = 2;

    pub fn clear_batch(&mut self) {
        self.chunk = ::std::option::Option::None;
    }

    pub fn has_batch(&self) -> bool {
        match self.chunk {
            ::std::option::Option::Some(WriteRequest_oneof_chunk::batch(..)) => true,
            _ => false,
        }
    }

    // Param is passed by value, moved
    pub fn set_batch(&mut self, v: WriteBatch) {
        self.chunk = ::std::option::Option::Some(WriteRequest_oneof_chunk::batch(v))
    }

    // Mutable pointer to the field.
    pub fn mut_batch(&mut self) -> &mut WriteBatch {
        if let ::std::option::Option::Some(WriteRequest_oneof_chunk::batch(_)) = self.chunk {
        } else {
            self.chunk = ::std::option::Option::Some(WriteRequest_oneof_chunk::batch(WriteBatch::new()));
        }
        match self.chunk {
            ::std::option::Option::Some(WriteRequest_oneof_chunk::batch(ref mut v)) => v,
            _ => panic!(),
        }
    }

    // Take field
    pub fn take_batch(&mut self) -> WriteBatch {
        if self.has_batch() {
            match self.chunk.take() {
                ::std::option::Option::Some(WriteRequest_oneof_chunk::batch(v)) => v,
                _ => panic!(),
            }
        } else {
            WriteBatch::new()
        }
    }

    pub fn get_batch(&self) -> &WriteBatch {
        match self.chunk {
            ::std::option::Option::Some(WriteRequest_oneof_chunk::batch(ref v)) => v,
            _ => WriteBatch::default_instance(),
        }
    }
}

impl ::protobuf::Message for WriteRequest {
    fn is_initialized(&self) -> bool {
        if let Some(WriteRequest_oneof_chunk::head(ref v)) = self.chunk {
            if !v.is_initialized() {
                return false;
            }
        }
        if let Some(WriteRequest_oneof_chunk::batch(ref v)) = self.chunk {
            if !v.is_initialized() {
                return false;
            }
        }
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    if wire_type != ::protobuf::wire_format::WireTypeLengthDelimited {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    self.chunk = ::std::option::Option::Some(WriteRequest_oneof_chunk::head(is.read_message()?));
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeLengthDelimited {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    self.chunk = ::std::option::Option::Some(WriteRequest_oneof_chunk::batch(is.read_message()?));
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if let ::std::option::Option::Some(ref v) = self.chunk {
            match v {
                &WriteRequest_oneof_chunk::head(ref v) => {
                    let len = v.compute_size();
                    my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
                },
                &WriteRequest_oneof_chunk::batch(ref v) => {
                    let len = v.compute_size();
                    my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
                },
            };
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let ::std::option::Option::Some(ref v) = self.chunk {
            match v {
                &WriteRequest_oneof_chunk::head(ref v) => {
                    os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
                    os.write_raw_varint32(v.get_cached_size())?;
                    v.write_to_with_cached_sizes(os)?;
                },
                &WriteRequest_oneof_chunk::batch(ref v) => {
                    os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
                    os.write_raw_varint32(v.get_cached_size())?;
                    v.write_to_with_cached_sizes(os)?;
                },
            };
        }
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for WriteRequest {
    fn new() -> WriteRequest {
        WriteRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<WriteRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_message_accessor::<_, WriteHead>(
                    "head",
                    WriteRequest::has_head,
                    WriteRequest::get_head,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_message_accessor::<_, WriteBatch>(
                    "batch",
                    WriteRequest::has_batch,
                    WriteRequest::get_batch,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<WriteRequest>(
                    "WriteRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for WriteRequest {
    fn clear(&mut self) {
        self.clear_head();
        self.clear_batch();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for WriteRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for WriteRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct WriteResponse {
    // message fields
    pub error: ::protobuf::SingularPtrField<Error>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for WriteResponse {}

impl WriteResponse {
    pub fn new() -> WriteResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static WriteResponse {
        static mut instance: ::protobuf::lazy::Lazy<WriteResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const WriteResponse,
        };
        unsafe {
            instance.get(WriteResponse::new)
        }
    }

    // .import_kvpb.Error error = 1;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: Error) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut Error {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> Error {
        self.error.take().unwrap_or_else(|| Error::new())
    }

    pub fn get_error(&self) -> &Error {
        self.error.as_ref().unwrap_or_else(|| Error::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<Error> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Error> {
        &mut self.error
    }
}

impl ::protobuf::Message for WriteResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.error {
            if !v.is_initialized() {
                return false;
            }
        };
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.error)?;
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for WriteResponse {
    fn new() -> WriteResponse {
        WriteResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<WriteResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Error>>(
                    "error",
                    WriteResponse::get_error_for_reflect,
                    WriteResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<WriteResponse>(
                    "WriteResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for WriteResponse {
    fn clear(&mut self) {
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for WriteResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for WriteResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct CloseRequest {
    // message fields
    pub uuid: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for CloseRequest {}

impl CloseRequest {
    pub fn new() -> CloseRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static CloseRequest {
        static mut instance: ::protobuf::lazy::Lazy<CloseRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const CloseRequest,
        };
        unsafe {
            instance.get(CloseRequest::new)
        }
    }

    // bytes uuid = 1;

    pub fn clear_uuid(&mut self) {
        self.uuid.clear();
    }

    // Param is passed by value, moved
    pub fn set_uuid(&mut self, v: ::std::vec::Vec<u8>) {
        self.uuid = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_uuid(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.uuid
    }

    // Take field
    pub fn take_uuid(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.uuid, ::std::vec::Vec::new())
    }

    pub fn get_uuid(&self) -> &[u8] {
        &self.uuid
    }

    fn get_uuid_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.uuid
    }

    fn mut_uuid_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.uuid
    }
}

impl ::protobuf::Message for CloseRequest {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.uuid)?;
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if !self.uuid.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.uuid);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.uuid.is_empty() {
            os.write_bytes(1, &self.uuid)?;
        }
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for CloseRequest {
    fn new() -> CloseRequest {
        CloseRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<CloseRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "uuid",
                    CloseRequest::get_uuid_for_reflect,
                    CloseRequest::mut_uuid_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<CloseRequest>(
                    "CloseRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for CloseRequest {
    fn clear(&mut self) {
        self.clear_uuid();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for CloseRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for CloseRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct CloseResponse {
    // message fields
    pub error: ::protobuf::SingularPtrField<Error>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for CloseResponse {}

impl CloseResponse {
    pub fn new() -> CloseResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static CloseResponse {
        static mut instance: ::protobuf::lazy::Lazy<CloseResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const CloseResponse,
        };
        unsafe {
            instance.get(CloseResponse::new)
        }
    }

    // .import_kvpb.Error error = 1;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: Error) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut Error {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> Error {
        self.error.take().unwrap_or_else(|| Error::new())
    }

    pub fn get_error(&self) -> &Error {
        self.error.as_ref().unwrap_or_else(|| Error::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<Error> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Error> {
        &mut self.error
    }
}

impl ::protobuf::Message for CloseResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.error {
            if !v.is_initialized() {
                return false;
            }
        };
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.error)?;
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for CloseResponse {
    fn new() -> CloseResponse {
        CloseResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<CloseResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Error>>(
                    "error",
                    CloseResponse::get_error_for_reflect,
                    CloseResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<CloseResponse>(
                    "CloseResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for CloseResponse {
    fn clear(&mut self) {
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for CloseResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for CloseResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct Error {
    // message fields
    pub engine_not_found: ::protobuf::SingularPtrField<Error_EngineNotFound>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for Error {}

impl Error {
    pub fn new() -> Error {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static Error {
        static mut instance: ::protobuf::lazy::Lazy<Error> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const Error,
        };
        unsafe {
            instance.get(Error::new)
        }
    }

    // .import_kvpb.Error.EngineNotFound engine_not_found = 1;

    pub fn clear_engine_not_found(&mut self) {
        self.engine_not_found.clear();
    }

    pub fn has_engine_not_found(&self) -> bool {
        self.engine_not_found.is_some()
    }

    // Param is passed by value, moved
    pub fn set_engine_not_found(&mut self, v: Error_EngineNotFound) {
        self.engine_not_found = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_engine_not_found(&mut self) -> &mut Error_EngineNotFound {
        if self.engine_not_found.is_none() {
            self.engine_not_found.set_default();
        }
        self.engine_not_found.as_mut().unwrap()
    }

    // Take field
    pub fn take_engine_not_found(&mut self) -> Error_EngineNotFound {
        self.engine_not_found.take().unwrap_or_else(|| Error_EngineNotFound::new())
    }

    pub fn get_engine_not_found(&self) -> &Error_EngineNotFound {
        self.engine_not_found.as_ref().unwrap_or_else(|| Error_EngineNotFound::default_instance())
    }

    fn get_engine_not_found_for_reflect(&self) -> &::protobuf::SingularPtrField<Error_EngineNotFound> {
        &self.engine_not_found
    }

    fn mut_engine_not_found_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Error_EngineNotFound> {
        &mut self.engine_not_found
    }
}

impl ::protobuf::Message for Error {
    fn is_initialized(&self) -> bool {
        for v in &self.engine_not_found {
            if !v.is_initialized() {
                return false;
            }
        };
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.engine_not_found)?;
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if let Some(ref v) = self.engine_not_found.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.engine_not_found.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for Error {
    fn new() -> Error {
        Error::new()
    }

    fn descriptor_static(_: ::std::option::Option<Error>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Error_EngineNotFound>>(
                    "engine_not_found",
                    Error::get_engine_not_found_for_reflect,
                    Error::mut_engine_not_found_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<Error>(
                    "Error",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for Error {
    fn clear(&mut self) {
        self.clear_engine_not_found();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for Error {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for Error {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct Error_EngineNotFound {
    // message fields
    pub uuid: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for Error_EngineNotFound {}

impl Error_EngineNotFound {
    pub fn new() -> Error_EngineNotFound {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static Error_EngineNotFound {
        static mut instance: ::protobuf::lazy::Lazy<Error_EngineNotFound> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const Error_EngineNotFound,
        };
        unsafe {
            instance.get(Error_EngineNotFound::new)
        }
    }

    // bytes uuid = 1;

    pub fn clear_uuid(&mut self) {
        self.uuid.clear();
    }

    // Param is passed by value, moved
    pub fn set_uuid(&mut self, v: ::std::vec::Vec<u8>) {
        self.uuid = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_uuid(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.uuid
    }

    // Take field
    pub fn take_uuid(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.uuid, ::std::vec::Vec::new())
    }

    pub fn get_uuid(&self) -> &[u8] {
        &self.uuid
    }

    fn get_uuid_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.uuid
    }

    fn mut_uuid_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.uuid
    }
}

impl ::protobuf::Message for Error_EngineNotFound {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.uuid)?;
                },
                _ => {
                    ::protobuf::rt::read_unknown_or_skip_group(field_number, wire_type, is, self.mut_unknown_fields())?;
                },
            };
        }
        ::std::result::Result::Ok(())
    }

    // Compute sizes of nested messages
    #[allow(unused_variables)]
    fn compute_size(&self) -> u32 {
        let mut my_size = 0;
        if !self.uuid.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.uuid);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.uuid.is_empty() {
            os.write_bytes(1, &self.uuid)?;
        }
        os.write_unknown_fields(self.get_unknown_fields())?;
        ::std::result::Result::Ok(())
    }

    fn get_cached_size(&self) -> u32 {
        self.cached_size.get()
    }

    fn get_unknown_fields(&self) -> &::protobuf::UnknownFields {
        &self.unknown_fields
    }

    fn mut_unknown_fields(&mut self) -> &mut ::protobuf::UnknownFields {
        &mut self.unknown_fields
    }

    fn as_any(&self) -> &::std::any::Any {
        self as &::std::any::Any
    }
    fn as_any_mut(&mut self) -> &mut ::std::any::Any {
        self as &mut ::std::any::Any
    }
    fn into_any(self: Box<Self>) -> ::std::boxed::Box<::std::any::Any> {
        self
    }

    fn descriptor(&self) -> &'static ::protobuf::reflect::MessageDescriptor {
        ::protobuf::MessageStatic::descriptor_static(None::<Self>)
    }
}

impl ::protobuf::MessageStatic for Error_EngineNotFound {
    fn new() -> Error_EngineNotFound {
        Error_EngineNotFound::new()
    }

    fn descriptor_static(_: ::std::option::Option<Error_EngineNotFound>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "uuid",
                    Error_EngineNotFound::get_uuid_for_reflect,
                    Error_EngineNotFound::mut_uuid_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<Error_EngineNotFound>(
                    "Error_EngineNotFound",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for Error_EngineNotFound {
    fn clear(&mut self) {
        self.clear_uuid();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for Error_EngineNotFound {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for Error_EngineNotFound {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

static file_descriptor_proto_data: &'static [u8] = b"\
    \n\x11import_kvpb.proto\x12\x0bimport_kvpb\x1a\x14gogoproto/gogo.proto\"\
    !\n\x0bOpenRequest\x12\x12\n\x04uuid\x18\x01\x20\x01(\x0cR\x04uuid\"\x0e\
    \n\x0cOpenResponse\"\x1f\n\tWriteHead\x12\x12\n\x04uuid\x18\x01\x20\x01(\
    \x0cR\x04uuid\"k\n\x08Mutation\x12(\n\x02op\x18\x01\x20\x01(\x0e2\x18.im\
    port_kvpb.Mutation.OPR\x02op\x12\x10\n\x03key\x18\x02\x20\x01(\x0cR\x03k\
    ey\x12\x14\n\x05value\x18\x03\x20\x01(\x0cR\x05value\"\r\n\x02OP\x12\x07\
    \n\x03Put\x10\0\"^\n\nWriteBatch\x12\x1b\n\tcommit_ts\x18\x01\x20\x01(\
    \x04R\x08commitTs\x123\n\tmutations\x18\x02\x20\x03(\x0b2\x15.import_kvp\
    b.MutationR\tmutations\"v\n\x0cWriteRequest\x12,\n\x04head\x18\x01\x20\
    \x01(\x0b2\x16.import_kvpb.WriteHeadH\0R\x04head\x12/\n\x05batch\x18\x02\
    \x20\x01(\x0b2\x17.import_kvpb.WriteBatchH\0R\x05batchB\x07\n\x05chunk\"\
    9\n\rWriteResponse\x12(\n\x05error\x18\x01\x20\x01(\x0b2\x12.import_kvpb\
    .ErrorR\x05error\"\"\n\x0cCloseRequest\x12\x12\n\x04uuid\x18\x01\x20\x01\
    (\x0cR\x04uuid\"9\n\rCloseResponse\x12(\n\x05error\x18\x01\x20\x01(\x0b2\
    \x12.import_kvpb.ErrorR\x05error\"z\n\x05Error\x12K\n\x10engine_not_foun\
    d\x18\x01\x20\x01(\x0b2!.import_kvpb.Error.EngineNotFoundR\x0eengineNotF\
    ound\x1a$\n\x0eEngineNotFound\x12\x12\n\x04uuid\x18\x01\x20\x01(\x0cR\
    \x04uuid2\xcf\x01\n\x08ImportKV\x12=\n\x04Open\x12\x18.import_kvpb.OpenR\
    equest\x1a\x19.import_kvpb.OpenResponse\"\0\x12B\n\x05Write\x12\x19.impo\
    rt_kvpb.WriteRequest\x1a\x1a.import_kvpb.WriteResponse\"\0(\x01\x12@\n\
    \x05Close\x12\x19.import_kvpb.CloseRequest\x1a\x1a.import_kvpb.CloseResp\
    onse\"\0B&\n\x18com.pingcap.tikv.kvproto\xd0\xe2\x1e\x01\xc8\xe2\x1e\x01\
    \xe0\xe2\x1e\x01J\xd0\x15\n\x06\x12\x04\0\0S\x01\n\x08\n\x01\x0c\x12\x03\
    \0\0\x12\n\x08\n\x01\x02\x12\x03\x02\x08\x13\n\t\n\x02\x03\0\x12\x03\x04\
    \x07\x1d\n\x08\n\x01\x08\x12\x03\x06\0$\n\x0b\n\x04\x08\xe7\x07\0\x12\
    \x03\x06\0$\n\x0c\n\x05\x08\xe7\x07\0\x02\x12\x03\x06\x07\x1c\n\r\n\x06\
    \x08\xe7\x07\0\x02\0\x12\x03\x06\x07\x1c\n\x0e\n\x07\x08\xe7\x07\0\x02\0\
    \x01\x12\x03\x06\x08\x1b\n\x0c\n\x05\x08\xe7\x07\0\x03\x12\x03\x06\x1f#\
    \n\x08\n\x01\x08\x12\x03\x07\0(\n\x0b\n\x04\x08\xe7\x07\x01\x12\x03\x07\
    \0(\n\x0c\n\x05\x08\xe7\x07\x01\x02\x12\x03\x07\x07\x20\n\r\n\x06\x08\
    \xe7\x07\x01\x02\0\x12\x03\x07\x07\x20\n\x0e\n\x07\x08\xe7\x07\x01\x02\0\
    \x01\x12\x03\x07\x08\x1f\n\x0c\n\x05\x08\xe7\x07\x01\x03\x12\x03\x07#'\n\
    \x08\n\x01\x08\x12\x03\x08\0*\n\x0b\n\x04\x08\xe7\x07\x02\x12\x03\x08\0*\
    \n\x0c\n\x05\x08\xe7\x07\x02\x02\x12\x03\x08\x07\"\n\r\n\x06\x08\xe7\x07\
    \x02\x02\0\x12\x03\x08\x07\"\n\x0e\n\x07\x08\xe7\x07\x02\x02\0\x01\x12\
    \x03\x08\x08!\n\x0c\n\x05\x08\xe7\x07\x02\x03\x12\x03\x08%)\n\x08\n\x01\
    \x08\x12\x03\n\01\n\x0b\n\x04\x08\xe7\x07\x03\x12\x03\n\01\n\x0c\n\x05\
    \x08\xe7\x07\x03\x02\x12\x03\n\x07\x13\n\r\n\x06\x08\xe7\x07\x03\x02\0\
    \x12\x03\n\x07\x13\n\x0e\n\x07\x08\xe7\x07\x03\x02\0\x01\x12\x03\n\x07\
    \x13\n\x0c\n\x05\x08\xe7\x07\x03\x07\x12\x03\n\x160\n\x9d\x04\n\x02\x06\
    \0\x12\x04\x15\0\x1c\x01\x1a\x90\x04\x20ImportKV\x20provides\x20a\x20ser\
    vice\x20to\x20import\x20key-value\x20pairs\x20to\x20TiKV.\n\n\x20In\x20o\
    rder\x20to\x20import\x20key-value\x20pairs\x20to\x20TiKV,\x20the\x20user\
    \x20should:\n\x201.\x20Open\x20an\x20engine\x20identified\x20by\x20an\
    \x20UUID.\n\x202.\x20Open\x20write\x20streams\x20to\x20write\x20key-valu\
    e\x20batch\x20to\x20the\x20opened\x20engine.\n\x20\x20\x20\x20Different\
    \x20streams/clients\x20can\x20write\x20to\x20the\x20same\x20engine\x20co\
    ncurrently.\n\x203.\x20Close\x20the\x20engine\x20after\x20all\x20write\
    \x20batches\x20are\x20finished.\x20An\x20engine\x20can\x20only\n\x20\x20\
    \x20\x20be\x20closed\x20when\x20all\x20write\x20streams\x20are\x20closed\
    .\x20An\x20engine\x20can\x20only\x20be\x20closed\n\x20\x20\x20\x20once,\
    \x20and\x20it\x20can\x20not\x20be\x20opened\x20again\x20once\x20it\x20is\
    \x20closed.\n\n\n\n\x03\x06\0\x01\x12\x03\x15\x08\x10\n\x1e\n\x04\x06\0\
    \x02\0\x12\x03\x17\x043\x1a\x11\x20Open\x20an\x20engine.\n\n\x0c\n\x05\
    \x06\0\x02\0\x01\x12\x03\x17\x08\x0c\n\x0c\n\x05\x06\0\x02\0\x02\x12\x03\
    \x17\r\x18\n\x0c\n\x05\x06\0\x02\0\x03\x12\x03\x17#/\n1\n\x04\x06\0\x02\
    \x01\x12\x03\x19\x04=\x1a$\x20Open\x20a\x20write\x20stream\x20to\x20the\
    \x20engine.\n\n\x0c\n\x05\x06\0\x02\x01\x01\x12\x03\x19\x08\r\n\x0c\n\
    \x05\x06\0\x02\x01\x05\x12\x03\x19\x0e\x14\n\x0c\n\x05\x06\0\x02\x01\x02\
    \x12\x03\x19\x15!\n\x0c\n\x05\x06\0\x02\x01\x03\x12\x03\x19,9\n\x20\n\
    \x04\x06\0\x02\x02\x12\x03\x1b\x046\x1a\x13\x20Close\x20the\x20engine.\n\
    \n\x0c\n\x05\x06\0\x02\x02\x01\x12\x03\x1b\x08\r\n\x0c\n\x05\x06\0\x02\
    \x02\x02\x12\x03\x1b\x0e\x1a\n\x0c\n\x05\x06\0\x02\x02\x03\x12\x03\x1b%2\
    \n\n\n\x02\x04\0\x12\x04\x1e\0\x20\x01\n\n\n\x03\x04\0\x01\x12\x03\x1e\
    \x08\x13\n\x0b\n\x04\x04\0\x02\0\x12\x03\x1f\x04\x13\n\r\n\x05\x04\0\x02\
    \0\x04\x12\x04\x1f\x04\x1e\x15\n\x0c\n\x05\x04\0\x02\0\x05\x12\x03\x1f\
    \x04\t\n\x0c\n\x05\x04\0\x02\0\x01\x12\x03\x1f\n\x0e\n\x0c\n\x05\x04\0\
    \x02\0\x03\x12\x03\x1f\x11\x12\n\n\n\x02\x04\x01\x12\x04\"\0#\x01\n\n\n\
    \x03\x04\x01\x01\x12\x03\"\x08\x14\n\n\n\x02\x04\x02\x12\x04%\0'\x01\n\n\
    \n\x03\x04\x02\x01\x12\x03%\x08\x11\n\x0b\n\x04\x04\x02\x02\0\x12\x03&\
    \x04\x13\n\r\n\x05\x04\x02\x02\0\x04\x12\x04&\x04%\x13\n\x0c\n\x05\x04\
    \x02\x02\0\x05\x12\x03&\x04\t\n\x0c\n\x05\x04\x02\x02\0\x01\x12\x03&\n\
    \x0e\n\x0c\n\x05\x04\x02\x02\0\x03\x12\x03&\x11\x12\n\n\n\x02\x04\x03\
    \x12\x04)\00\x01\n\n\n\x03\x04\x03\x01\x12\x03)\x08\x10\n\x0c\n\x04\x04\
    \x03\x04\0\x12\x04*\x04,\x05\n\x0c\n\x05\x04\x03\x04\0\x01\x12\x03*\t\
    \x0b\n\r\n\x06\x04\x03\x04\0\x02\0\x12\x03+\x08\x10\n\x0e\n\x07\x04\x03\
    \x04\0\x02\0\x01\x12\x03+\x08\x0b\n\x0e\n\x07\x04\x03\x04\0\x02\0\x02\
    \x12\x03+\x0e\x0f\n\x0b\n\x04\x04\x03\x02\0\x12\x03-\x04\x0e\n\r\n\x05\
    \x04\x03\x02\0\x04\x12\x04-\x04,\x05\n\x0c\n\x05\x04\x03\x02\0\x06\x12\
    \x03-\x04\x06\n\x0c\n\x05\x04\x03\x02\0\x01\x12\x03-\x07\t\n\x0c\n\x05\
    \x04\x03\x02\0\x03\x12\x03-\x0c\r\n\x0b\n\x04\x04\x03\x02\x01\x12\x03.\
    \x04\x12\n\r\n\x05\x04\x03\x02\x01\x04\x12\x04.\x04-\x0e\n\x0c\n\x05\x04\
    \x03\x02\x01\x05\x12\x03.\x04\t\n\x0c\n\x05\x04\x03\x02\x01\x01\x12\x03.\
    \n\r\n\x0c\n\x05\x04\x03\x02\x01\x03\x12\x03.\x10\x11\n\x0b\n\x04\x04\
    \x03\x02\x02\x12\x03/\x04\x14\n\r\n\x05\x04\x03\x02\x02\x04\x12\x04/\x04\
    .\x12\n\x0c\n\x05\x04\x03\x02\x02\x05\x12\x03/\x04\t\n\x0c\n\x05\x04\x03\
    \x02\x02\x01\x12\x03/\n\x0f\n\x0c\n\x05\x04\x03\x02\x02\x03\x12\x03/\x12\
    \x13\n\n\n\x02\x04\x04\x12\x042\05\x01\n\n\n\x03\x04\x04\x01\x12\x032\
    \x08\x12\n\x0b\n\x04\x04\x04\x02\0\x12\x033\x04\x19\n\r\n\x05\x04\x04\
    \x02\0\x04\x12\x043\x042\x14\n\x0c\n\x05\x04\x04\x02\0\x05\x12\x033\x04\
    \n\n\x0c\n\x05\x04\x04\x02\0\x01\x12\x033\x0b\x14\n\x0c\n\x05\x04\x04\
    \x02\0\x03\x12\x033\x17\x18\n\x0b\n\x04\x04\x04\x02\x01\x12\x034\x04$\n\
    \x0c\n\x05\x04\x04\x02\x01\x04\x12\x034\x04\x0c\n\x0c\n\x05\x04\x04\x02\
    \x01\x06\x12\x034\r\x15\n\x0c\n\x05\x04\x04\x02\x01\x01\x12\x034\x16\x1f\
    \n\x0c\n\x05\x04\x04\x02\x01\x03\x12\x034\"#\n\n\n\x02\x04\x05\x12\x047\
    \0<\x01\n\n\n\x03\x04\x05\x01\x12\x037\x08\x14\n\x0c\n\x04\x04\x05\x08\0\
    \x12\x048\x04;\x05\n\x0c\n\x05\x04\x05\x08\0\x01\x12\x038\n\x0f\n\x0b\n\
    \x04\x04\x05\x02\0\x12\x039\x08\x1b\n\x0c\n\x05\x04\x05\x02\0\x06\x12\
    \x039\x08\x11\n\x0c\n\x05\x04\x05\x02\0\x01\x12\x039\x12\x16\n\x0c\n\x05\
    \x04\x05\x02\0\x03\x12\x039\x19\x1a\n\x0b\n\x04\x04\x05\x02\x01\x12\x03:\
    \x08\x1d\n\x0c\n\x05\x04\x05\x02\x01\x06\x12\x03:\x08\x12\n\x0c\n\x05\
    \x04\x05\x02\x01\x01\x12\x03:\x13\x18\n\x0c\n\x05\x04\x05\x02\x01\x03\
    \x12\x03:\x1b\x1c\n\n\n\x02\x04\x06\x12\x04>\0@\x01\n\n\n\x03\x04\x06\
    \x01\x12\x03>\x08\x15\n\x0b\n\x04\x04\x06\x02\0\x12\x03?\x04\x14\n\r\n\
    \x05\x04\x06\x02\0\x04\x12\x04?\x04>\x17\n\x0c\n\x05\x04\x06\x02\0\x06\
    \x12\x03?\x04\t\n\x0c\n\x05\x04\x06\x02\0\x01\x12\x03?\n\x0f\n\x0c\n\x05\
    \x04\x06\x02\0\x03\x12\x03?\x12\x13\n\n\n\x02\x04\x07\x12\x04B\0D\x01\n\
    \n\n\x03\x04\x07\x01\x12\x03B\x08\x14\n\x0b\n\x04\x04\x07\x02\0\x12\x03C\
    \x04\x13\n\r\n\x05\x04\x07\x02\0\x04\x12\x04C\x04B\x16\n\x0c\n\x05\x04\
    \x07\x02\0\x05\x12\x03C\x04\t\n\x0c\n\x05\x04\x07\x02\0\x01\x12\x03C\n\
    \x0e\n\x0c\n\x05\x04\x07\x02\0\x03\x12\x03C\x11\x12\n\n\n\x02\x04\x08\
    \x12\x04F\0H\x01\n\n\n\x03\x04\x08\x01\x12\x03F\x08\x15\n\x0b\n\x04\x04\
    \x08\x02\0\x12\x03G\x04\x14\n\r\n\x05\x04\x08\x02\0\x04\x12\x04G\x04F\
    \x17\n\x0c\n\x05\x04\x08\x02\0\x06\x12\x03G\x04\t\n\x0c\n\x05\x04\x08\
    \x02\0\x01\x12\x03G\n\x0f\n\x0c\n\x05\x04\x08\x02\0\x03\x12\x03G\x12\x13\
    \n\n\n\x02\x04\t\x12\x04J\0S\x01\n\n\n\x03\x04\t\x01\x12\x03J\x08\r\n\
    \x0c\n\x04\x04\t\x03\0\x12\x04K\x04M\x05\n\x0c\n\x05\x04\t\x03\0\x01\x12\
    \x03K\x0c\x1a\n\r\n\x06\x04\t\x03\0\x02\0\x12\x03L\x08\x17\n\x0f\n\x07\
    \x04\t\x03\0\x02\0\x04\x12\x04L\x08K\x1c\n\x0e\n\x07\x04\t\x03\0\x02\0\
    \x05\x12\x03L\x08\r\n\x0e\n\x07\x04\t\x03\0\x02\0\x01\x12\x03L\x0e\x12\n\
    \x0e\n\x07\x04\t\x03\0\x02\0\x03\x12\x03L\x15\x16\n\x8c\x02\n\x04\x04\t\
    \x02\0\x12\x03R\x04(\x1a\xfe\x01\x20This\x20can\x20happen\x20if\x20the\
    \x20client\x20hasn't\x20opened\x20the\x20engine,\x20or\x20the\x20server\
    \n\x20restarts\x20while\x20the\x20client\x20is\x20writing\x20or\x20closi\
    ng.\x20An\x20unclosed\x20engine\x20will\n\x20be\x20removed\x20on\x20serv\
    er\x20restart,\x20so\x20the\x20client\x20should\x20not\x20continue\x20bu\
    t\n\x20restart\x20the\x20previous\x20job\x20in\x20that\x20case.\n\n\r\n\
    \x05\x04\t\x02\0\x04\x12\x04R\x04M\x05\n\x0c\n\x05\x04\t\x02\0\x06\x12\
    \x03R\x04\x12\n\x0c\n\x05\x04\t\x02\0\x01\x12\x03R\x13#\n\x0c\n\x05\x04\
    \t\x02\0\x03\x12\x03R&'b\x06proto3\
";

static mut file_descriptor_proto_lazy: ::protobuf::lazy::Lazy<::protobuf::descriptor::FileDescriptorProto> = ::protobuf::lazy::Lazy {
    lock: ::protobuf::lazy::ONCE_INIT,
    ptr: 0 as *const ::protobuf::descriptor::FileDescriptorProto,
};

fn parse_descriptor_proto() -> ::protobuf::descriptor::FileDescriptorProto {
    ::protobuf::parse_from_bytes(file_descriptor_proto_data).unwrap()
}

pub fn file_descriptor_proto() -> &'static ::protobuf::descriptor::FileDescriptorProto {
    unsafe {
        file_descriptor_proto_lazy.get(|| {
            parse_descriptor_proto()
        })
    }
}
