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
pub struct GetRequest {
    // message fields
    pub db: DB,
    pub cf: ::std::string::String,
    pub key: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for GetRequest {}

impl GetRequest {
    pub fn new() -> GetRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static GetRequest {
        static mut instance: ::protobuf::lazy::Lazy<GetRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const GetRequest,
        };
        unsafe {
            instance.get(GetRequest::new)
        }
    }

    // .debugpb.DB db = 1;

    pub fn clear_db(&mut self) {
        self.db = DB::INVALID;
    }

    // Param is passed by value, moved
    pub fn set_db(&mut self, v: DB) {
        self.db = v;
    }

    pub fn get_db(&self) -> DB {
        self.db
    }

    fn get_db_for_reflect(&self) -> &DB {
        &self.db
    }

    fn mut_db_for_reflect(&mut self) -> &mut DB {
        &mut self.db
    }

    // string cf = 2;

    pub fn clear_cf(&mut self) {
        self.cf.clear();
    }

    // Param is passed by value, moved
    pub fn set_cf(&mut self, v: ::std::string::String) {
        self.cf = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_cf(&mut self) -> &mut ::std::string::String {
        &mut self.cf
    }

    // Take field
    pub fn take_cf(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.cf, ::std::string::String::new())
    }

    pub fn get_cf(&self) -> &str {
        &self.cf
    }

    fn get_cf_for_reflect(&self) -> &::std::string::String {
        &self.cf
    }

    fn mut_cf_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.cf
    }

    // bytes key = 3;

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
}

impl ::protobuf::Message for GetRequest {
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
                    self.db = tmp;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
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
        if self.db != DB::INVALID {
            my_size += ::protobuf::rt::enum_size(1, self.db);
        }
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.cf);
        }
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.key);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.db != DB::INVALID {
            os.write_enum(1, self.db.value())?;
        }
        if !self.cf.is_empty() {
            os.write_string(2, &self.cf)?;
        }
        if !self.key.is_empty() {
            os.write_bytes(3, &self.key)?;
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

impl ::protobuf::MessageStatic for GetRequest {
    fn new() -> GetRequest {
        GetRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<GetRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeEnum<DB>>(
                    "db",
                    GetRequest::get_db_for_reflect,
                    GetRequest::mut_db_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    GetRequest::get_cf_for_reflect,
                    GetRequest::mut_cf_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    GetRequest::get_key_for_reflect,
                    GetRequest::mut_key_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<GetRequest>(
                    "GetRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for GetRequest {
    fn clear(&mut self) {
        self.clear_db();
        self.clear_cf();
        self.clear_key();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for GetRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for GetRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct GetResponse {
    // message fields
    pub value: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for GetResponse {}

impl GetResponse {
    pub fn new() -> GetResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static GetResponse {
        static mut instance: ::protobuf::lazy::Lazy<GetResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const GetResponse,
        };
        unsafe {
            instance.get(GetResponse::new)
        }
    }

    // bytes value = 1;

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

impl ::protobuf::Message for GetResponse {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
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
        if !self.value.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.value);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.value.is_empty() {
            os.write_bytes(1, &self.value)?;
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

impl ::protobuf::MessageStatic for GetResponse {
    fn new() -> GetResponse {
        GetResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<GetResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "value",
                    GetResponse::get_value_for_reflect,
                    GetResponse::mut_value_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<GetResponse>(
                    "GetResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for GetResponse {
    fn clear(&mut self) {
        self.clear_value();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for GetResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for GetResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RaftLogRequest {
    // message fields
    pub region_id: u64,
    pub log_index: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RaftLogRequest {}

impl RaftLogRequest {
    pub fn new() -> RaftLogRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RaftLogRequest {
        static mut instance: ::protobuf::lazy::Lazy<RaftLogRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RaftLogRequest,
        };
        unsafe {
            instance.get(RaftLogRequest::new)
        }
    }

    // uint64 region_id = 1;

    pub fn clear_region_id(&mut self) {
        self.region_id = 0;
    }

    // Param is passed by value, moved
    pub fn set_region_id(&mut self, v: u64) {
        self.region_id = v;
    }

    pub fn get_region_id(&self) -> u64 {
        self.region_id
    }

    fn get_region_id_for_reflect(&self) -> &u64 {
        &self.region_id
    }

    fn mut_region_id_for_reflect(&mut self) -> &mut u64 {
        &mut self.region_id
    }

    // uint64 log_index = 2;

    pub fn clear_log_index(&mut self) {
        self.log_index = 0;
    }

    // Param is passed by value, moved
    pub fn set_log_index(&mut self, v: u64) {
        self.log_index = v;
    }

    pub fn get_log_index(&self) -> u64 {
        self.log_index
    }

    fn get_log_index_for_reflect(&self) -> &u64 {
        &self.log_index
    }

    fn mut_log_index_for_reflect(&mut self) -> &mut u64 {
        &mut self.log_index
    }
}

impl ::protobuf::Message for RaftLogRequest {
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
                    let tmp = is.read_uint64()?;
                    self.region_id = tmp;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.log_index = tmp;
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
        if self.region_id != 0 {
            my_size += ::protobuf::rt::value_size(1, self.region_id, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.log_index != 0 {
            my_size += ::protobuf::rt::value_size(2, self.log_index, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.region_id != 0 {
            os.write_uint64(1, self.region_id)?;
        }
        if self.log_index != 0 {
            os.write_uint64(2, self.log_index)?;
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

impl ::protobuf::MessageStatic for RaftLogRequest {
    fn new() -> RaftLogRequest {
        RaftLogRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RaftLogRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "region_id",
                    RaftLogRequest::get_region_id_for_reflect,
                    RaftLogRequest::mut_region_id_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "log_index",
                    RaftLogRequest::get_log_index_for_reflect,
                    RaftLogRequest::mut_log_index_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RaftLogRequest>(
                    "RaftLogRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RaftLogRequest {
    fn clear(&mut self) {
        self.clear_region_id();
        self.clear_log_index();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RaftLogRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RaftLogRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RaftLogResponse {
    // message fields
    pub entry: ::protobuf::SingularPtrField<super::eraftpb::Entry>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RaftLogResponse {}

impl RaftLogResponse {
    pub fn new() -> RaftLogResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RaftLogResponse {
        static mut instance: ::protobuf::lazy::Lazy<RaftLogResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RaftLogResponse,
        };
        unsafe {
            instance.get(RaftLogResponse::new)
        }
    }

    // .eraftpb.Entry entry = 1;

    pub fn clear_entry(&mut self) {
        self.entry.clear();
    }

    pub fn has_entry(&self) -> bool {
        self.entry.is_some()
    }

    // Param is passed by value, moved
    pub fn set_entry(&mut self, v: super::eraftpb::Entry) {
        self.entry = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_entry(&mut self) -> &mut super::eraftpb::Entry {
        if self.entry.is_none() {
            self.entry.set_default();
        }
        self.entry.as_mut().unwrap()
    }

    // Take field
    pub fn take_entry(&mut self) -> super::eraftpb::Entry {
        self.entry.take().unwrap_or_else(|| super::eraftpb::Entry::new())
    }

    pub fn get_entry(&self) -> &super::eraftpb::Entry {
        self.entry.as_ref().unwrap_or_else(|| super::eraftpb::Entry::default_instance())
    }

    fn get_entry_for_reflect(&self) -> &::protobuf::SingularPtrField<super::eraftpb::Entry> {
        &self.entry
    }

    fn mut_entry_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::eraftpb::Entry> {
        &mut self.entry
    }
}

impl ::protobuf::Message for RaftLogResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.entry {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.entry)?;
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
        if let Some(ref v) = self.entry.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.entry.as_ref() {
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

impl ::protobuf::MessageStatic for RaftLogResponse {
    fn new() -> RaftLogResponse {
        RaftLogResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RaftLogResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::eraftpb::Entry>>(
                    "entry",
                    RaftLogResponse::get_entry_for_reflect,
                    RaftLogResponse::mut_entry_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RaftLogResponse>(
                    "RaftLogResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RaftLogResponse {
    fn clear(&mut self) {
        self.clear_entry();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RaftLogResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RaftLogResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RegionInfoRequest {
    // message fields
    pub region_id: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RegionInfoRequest {}

impl RegionInfoRequest {
    pub fn new() -> RegionInfoRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RegionInfoRequest {
        static mut instance: ::protobuf::lazy::Lazy<RegionInfoRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RegionInfoRequest,
        };
        unsafe {
            instance.get(RegionInfoRequest::new)
        }
    }

    // uint64 region_id = 1;

    pub fn clear_region_id(&mut self) {
        self.region_id = 0;
    }

    // Param is passed by value, moved
    pub fn set_region_id(&mut self, v: u64) {
        self.region_id = v;
    }

    pub fn get_region_id(&self) -> u64 {
        self.region_id
    }

    fn get_region_id_for_reflect(&self) -> &u64 {
        &self.region_id
    }

    fn mut_region_id_for_reflect(&mut self) -> &mut u64 {
        &mut self.region_id
    }
}

impl ::protobuf::Message for RegionInfoRequest {
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
                    let tmp = is.read_uint64()?;
                    self.region_id = tmp;
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
        if self.region_id != 0 {
            my_size += ::protobuf::rt::value_size(1, self.region_id, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.region_id != 0 {
            os.write_uint64(1, self.region_id)?;
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

impl ::protobuf::MessageStatic for RegionInfoRequest {
    fn new() -> RegionInfoRequest {
        RegionInfoRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RegionInfoRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "region_id",
                    RegionInfoRequest::get_region_id_for_reflect,
                    RegionInfoRequest::mut_region_id_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RegionInfoRequest>(
                    "RegionInfoRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RegionInfoRequest {
    fn clear(&mut self) {
        self.clear_region_id();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RegionInfoRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RegionInfoRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RegionInfoResponse {
    // message fields
    pub raft_local_state: ::protobuf::SingularPtrField<super::raft_serverpb::RaftLocalState>,
    pub raft_apply_state: ::protobuf::SingularPtrField<super::raft_serverpb::RaftApplyState>,
    pub region_local_state: ::protobuf::SingularPtrField<super::raft_serverpb::RegionLocalState>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RegionInfoResponse {}

impl RegionInfoResponse {
    pub fn new() -> RegionInfoResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RegionInfoResponse {
        static mut instance: ::protobuf::lazy::Lazy<RegionInfoResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RegionInfoResponse,
        };
        unsafe {
            instance.get(RegionInfoResponse::new)
        }
    }

    // .raft_serverpb.RaftLocalState raft_local_state = 1;

    pub fn clear_raft_local_state(&mut self) {
        self.raft_local_state.clear();
    }

    pub fn has_raft_local_state(&self) -> bool {
        self.raft_local_state.is_some()
    }

    // Param is passed by value, moved
    pub fn set_raft_local_state(&mut self, v: super::raft_serverpb::RaftLocalState) {
        self.raft_local_state = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_raft_local_state(&mut self) -> &mut super::raft_serverpb::RaftLocalState {
        if self.raft_local_state.is_none() {
            self.raft_local_state.set_default();
        }
        self.raft_local_state.as_mut().unwrap()
    }

    // Take field
    pub fn take_raft_local_state(&mut self) -> super::raft_serverpb::RaftLocalState {
        self.raft_local_state.take().unwrap_or_else(|| super::raft_serverpb::RaftLocalState::new())
    }

    pub fn get_raft_local_state(&self) -> &super::raft_serverpb::RaftLocalState {
        self.raft_local_state.as_ref().unwrap_or_else(|| super::raft_serverpb::RaftLocalState::default_instance())
    }

    fn get_raft_local_state_for_reflect(&self) -> &::protobuf::SingularPtrField<super::raft_serverpb::RaftLocalState> {
        &self.raft_local_state
    }

    fn mut_raft_local_state_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::raft_serverpb::RaftLocalState> {
        &mut self.raft_local_state
    }

    // .raft_serverpb.RaftApplyState raft_apply_state = 2;

    pub fn clear_raft_apply_state(&mut self) {
        self.raft_apply_state.clear();
    }

    pub fn has_raft_apply_state(&self) -> bool {
        self.raft_apply_state.is_some()
    }

    // Param is passed by value, moved
    pub fn set_raft_apply_state(&mut self, v: super::raft_serverpb::RaftApplyState) {
        self.raft_apply_state = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_raft_apply_state(&mut self) -> &mut super::raft_serverpb::RaftApplyState {
        if self.raft_apply_state.is_none() {
            self.raft_apply_state.set_default();
        }
        self.raft_apply_state.as_mut().unwrap()
    }

    // Take field
    pub fn take_raft_apply_state(&mut self) -> super::raft_serverpb::RaftApplyState {
        self.raft_apply_state.take().unwrap_or_else(|| super::raft_serverpb::RaftApplyState::new())
    }

    pub fn get_raft_apply_state(&self) -> &super::raft_serverpb::RaftApplyState {
        self.raft_apply_state.as_ref().unwrap_or_else(|| super::raft_serverpb::RaftApplyState::default_instance())
    }

    fn get_raft_apply_state_for_reflect(&self) -> &::protobuf::SingularPtrField<super::raft_serverpb::RaftApplyState> {
        &self.raft_apply_state
    }

    fn mut_raft_apply_state_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::raft_serverpb::RaftApplyState> {
        &mut self.raft_apply_state
    }

    // .raft_serverpb.RegionLocalState region_local_state = 3;

    pub fn clear_region_local_state(&mut self) {
        self.region_local_state.clear();
    }

    pub fn has_region_local_state(&self) -> bool {
        self.region_local_state.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_local_state(&mut self, v: super::raft_serverpb::RegionLocalState) {
        self.region_local_state = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_local_state(&mut self) -> &mut super::raft_serverpb::RegionLocalState {
        if self.region_local_state.is_none() {
            self.region_local_state.set_default();
        }
        self.region_local_state.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_local_state(&mut self) -> super::raft_serverpb::RegionLocalState {
        self.region_local_state.take().unwrap_or_else(|| super::raft_serverpb::RegionLocalState::new())
    }

    pub fn get_region_local_state(&self) -> &super::raft_serverpb::RegionLocalState {
        self.region_local_state.as_ref().unwrap_or_else(|| super::raft_serverpb::RegionLocalState::default_instance())
    }

    fn get_region_local_state_for_reflect(&self) -> &::protobuf::SingularPtrField<super::raft_serverpb::RegionLocalState> {
        &self.region_local_state
    }

    fn mut_region_local_state_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::raft_serverpb::RegionLocalState> {
        &mut self.region_local_state
    }
}

impl ::protobuf::Message for RegionInfoResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.raft_local_state {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.raft_apply_state {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.region_local_state {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.raft_local_state)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.raft_apply_state)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_local_state)?;
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
        if let Some(ref v) = self.raft_local_state.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.raft_apply_state.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.region_local_state.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.raft_local_state.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.raft_apply_state.as_ref() {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.region_local_state.as_ref() {
            os.write_tag(3, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for RegionInfoResponse {
    fn new() -> RegionInfoResponse {
        RegionInfoResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RegionInfoResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::raft_serverpb::RaftLocalState>>(
                    "raft_local_state",
                    RegionInfoResponse::get_raft_local_state_for_reflect,
                    RegionInfoResponse::mut_raft_local_state_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::raft_serverpb::RaftApplyState>>(
                    "raft_apply_state",
                    RegionInfoResponse::get_raft_apply_state_for_reflect,
                    RegionInfoResponse::mut_raft_apply_state_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::raft_serverpb::RegionLocalState>>(
                    "region_local_state",
                    RegionInfoResponse::get_region_local_state_for_reflect,
                    RegionInfoResponse::mut_region_local_state_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RegionInfoResponse>(
                    "RegionInfoResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RegionInfoResponse {
    fn clear(&mut self) {
        self.clear_raft_local_state();
        self.clear_raft_apply_state();
        self.clear_region_local_state();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RegionInfoResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RegionInfoResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RegionSizeRequest {
    // message fields
    pub region_id: u64,
    pub cfs: ::protobuf::RepeatedField<::std::string::String>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RegionSizeRequest {}

impl RegionSizeRequest {
    pub fn new() -> RegionSizeRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RegionSizeRequest {
        static mut instance: ::protobuf::lazy::Lazy<RegionSizeRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RegionSizeRequest,
        };
        unsafe {
            instance.get(RegionSizeRequest::new)
        }
    }

    // uint64 region_id = 1;

    pub fn clear_region_id(&mut self) {
        self.region_id = 0;
    }

    // Param is passed by value, moved
    pub fn set_region_id(&mut self, v: u64) {
        self.region_id = v;
    }

    pub fn get_region_id(&self) -> u64 {
        self.region_id
    }

    fn get_region_id_for_reflect(&self) -> &u64 {
        &self.region_id
    }

    fn mut_region_id_for_reflect(&mut self) -> &mut u64 {
        &mut self.region_id
    }

    // repeated string cfs = 2;

    pub fn clear_cfs(&mut self) {
        self.cfs.clear();
    }

    // Param is passed by value, moved
    pub fn set_cfs(&mut self, v: ::protobuf::RepeatedField<::std::string::String>) {
        self.cfs = v;
    }

    // Mutable pointer to the field.
    pub fn mut_cfs(&mut self) -> &mut ::protobuf::RepeatedField<::std::string::String> {
        &mut self.cfs
    }

    // Take field
    pub fn take_cfs(&mut self) -> ::protobuf::RepeatedField<::std::string::String> {
        ::std::mem::replace(&mut self.cfs, ::protobuf::RepeatedField::new())
    }

    pub fn get_cfs(&self) -> &[::std::string::String] {
        &self.cfs
    }

    fn get_cfs_for_reflect(&self) -> &::protobuf::RepeatedField<::std::string::String> {
        &self.cfs
    }

    fn mut_cfs_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<::std::string::String> {
        &mut self.cfs
    }
}

impl ::protobuf::Message for RegionSizeRequest {
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
                    let tmp = is.read_uint64()?;
                    self.region_id = tmp;
                },
                2 => {
                    ::protobuf::rt::read_repeated_string_into(wire_type, is, &mut self.cfs)?;
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
        if self.region_id != 0 {
            my_size += ::protobuf::rt::value_size(1, self.region_id, ::protobuf::wire_format::WireTypeVarint);
        }
        for value in &self.cfs {
            my_size += ::protobuf::rt::string_size(2, &value);
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.region_id != 0 {
            os.write_uint64(1, self.region_id)?;
        }
        for v in &self.cfs {
            os.write_string(2, &v)?;
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

impl ::protobuf::MessageStatic for RegionSizeRequest {
    fn new() -> RegionSizeRequest {
        RegionSizeRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RegionSizeRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "region_id",
                    RegionSizeRequest::get_region_id_for_reflect,
                    RegionSizeRequest::mut_region_id_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cfs",
                    RegionSizeRequest::get_cfs_for_reflect,
                    RegionSizeRequest::mut_cfs_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RegionSizeRequest>(
                    "RegionSizeRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RegionSizeRequest {
    fn clear(&mut self) {
        self.clear_region_id();
        self.clear_cfs();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RegionSizeRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RegionSizeRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RegionSizeResponse {
    // message fields
    pub entries: ::protobuf::RepeatedField<RegionSizeResponse_Entry>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RegionSizeResponse {}

impl RegionSizeResponse {
    pub fn new() -> RegionSizeResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RegionSizeResponse {
        static mut instance: ::protobuf::lazy::Lazy<RegionSizeResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RegionSizeResponse,
        };
        unsafe {
            instance.get(RegionSizeResponse::new)
        }
    }

    // repeated .debugpb.RegionSizeResponse.Entry entries = 1;

    pub fn clear_entries(&mut self) {
        self.entries.clear();
    }

    // Param is passed by value, moved
    pub fn set_entries(&mut self, v: ::protobuf::RepeatedField<RegionSizeResponse_Entry>) {
        self.entries = v;
    }

    // Mutable pointer to the field.
    pub fn mut_entries(&mut self) -> &mut ::protobuf::RepeatedField<RegionSizeResponse_Entry> {
        &mut self.entries
    }

    // Take field
    pub fn take_entries(&mut self) -> ::protobuf::RepeatedField<RegionSizeResponse_Entry> {
        ::std::mem::replace(&mut self.entries, ::protobuf::RepeatedField::new())
    }

    pub fn get_entries(&self) -> &[RegionSizeResponse_Entry] {
        &self.entries
    }

    fn get_entries_for_reflect(&self) -> &::protobuf::RepeatedField<RegionSizeResponse_Entry> {
        &self.entries
    }

    fn mut_entries_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<RegionSizeResponse_Entry> {
        &mut self.entries
    }
}

impl ::protobuf::Message for RegionSizeResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.entries {
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
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.entries)?;
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
        for value in &self.entries {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        for v in &self.entries {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for RegionSizeResponse {
    fn new() -> RegionSizeResponse {
        RegionSizeResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RegionSizeResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<RegionSizeResponse_Entry>>(
                    "entries",
                    RegionSizeResponse::get_entries_for_reflect,
                    RegionSizeResponse::mut_entries_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RegionSizeResponse>(
                    "RegionSizeResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RegionSizeResponse {
    fn clear(&mut self) {
        self.clear_entries();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RegionSizeResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RegionSizeResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RegionSizeResponse_Entry {
    // message fields
    pub cf: ::std::string::String,
    pub size: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RegionSizeResponse_Entry {}

impl RegionSizeResponse_Entry {
    pub fn new() -> RegionSizeResponse_Entry {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RegionSizeResponse_Entry {
        static mut instance: ::protobuf::lazy::Lazy<RegionSizeResponse_Entry> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RegionSizeResponse_Entry,
        };
        unsafe {
            instance.get(RegionSizeResponse_Entry::new)
        }
    }

    // string cf = 1;

    pub fn clear_cf(&mut self) {
        self.cf.clear();
    }

    // Param is passed by value, moved
    pub fn set_cf(&mut self, v: ::std::string::String) {
        self.cf = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_cf(&mut self) -> &mut ::std::string::String {
        &mut self.cf
    }

    // Take field
    pub fn take_cf(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.cf, ::std::string::String::new())
    }

    pub fn get_cf(&self) -> &str {
        &self.cf
    }

    fn get_cf_for_reflect(&self) -> &::std::string::String {
        &self.cf
    }

    fn mut_cf_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.cf
    }

    // uint64 size = 2;

    pub fn clear_size(&mut self) {
        self.size = 0;
    }

    // Param is passed by value, moved
    pub fn set_size(&mut self, v: u64) {
        self.size = v;
    }

    pub fn get_size(&self) -> u64 {
        self.size
    }

    fn get_size_for_reflect(&self) -> &u64 {
        &self.size
    }

    fn mut_size_for_reflect(&mut self) -> &mut u64 {
        &mut self.size
    }
}

impl ::protobuf::Message for RegionSizeResponse_Entry {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.size = tmp;
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
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(1, &self.cf);
        }
        if self.size != 0 {
            my_size += ::protobuf::rt::value_size(2, self.size, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.cf.is_empty() {
            os.write_string(1, &self.cf)?;
        }
        if self.size != 0 {
            os.write_uint64(2, self.size)?;
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

impl ::protobuf::MessageStatic for RegionSizeResponse_Entry {
    fn new() -> RegionSizeResponse_Entry {
        RegionSizeResponse_Entry::new()
    }

    fn descriptor_static(_: ::std::option::Option<RegionSizeResponse_Entry>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RegionSizeResponse_Entry::get_cf_for_reflect,
                    RegionSizeResponse_Entry::mut_cf_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "size",
                    RegionSizeResponse_Entry::get_size_for_reflect,
                    RegionSizeResponse_Entry::mut_size_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RegionSizeResponse_Entry>(
                    "RegionSizeResponse_Entry",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RegionSizeResponse_Entry {
    fn clear(&mut self) {
        self.clear_cf();
        self.clear_size();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RegionSizeResponse_Entry {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RegionSizeResponse_Entry {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ScanMvccRequest {
    // message fields
    pub from_key: ::std::vec::Vec<u8>,
    pub to_key: ::std::vec::Vec<u8>,
    pub limit: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ScanMvccRequest {}

impl ScanMvccRequest {
    pub fn new() -> ScanMvccRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ScanMvccRequest {
        static mut instance: ::protobuf::lazy::Lazy<ScanMvccRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ScanMvccRequest,
        };
        unsafe {
            instance.get(ScanMvccRequest::new)
        }
    }

    // bytes from_key = 1;

    pub fn clear_from_key(&mut self) {
        self.from_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_from_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.from_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_from_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.from_key
    }

    // Take field
    pub fn take_from_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.from_key, ::std::vec::Vec::new())
    }

    pub fn get_from_key(&self) -> &[u8] {
        &self.from_key
    }

    fn get_from_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.from_key
    }

    fn mut_from_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.from_key
    }

    // bytes to_key = 2;

    pub fn clear_to_key(&mut self) {
        self.to_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_to_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.to_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_to_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.to_key
    }

    // Take field
    pub fn take_to_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.to_key, ::std::vec::Vec::new())
    }

    pub fn get_to_key(&self) -> &[u8] {
        &self.to_key
    }

    fn get_to_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.to_key
    }

    fn mut_to_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.to_key
    }

    // uint64 limit = 3;

    pub fn clear_limit(&mut self) {
        self.limit = 0;
    }

    // Param is passed by value, moved
    pub fn set_limit(&mut self, v: u64) {
        self.limit = v;
    }

    pub fn get_limit(&self) -> u64 {
        self.limit
    }

    fn get_limit_for_reflect(&self) -> &u64 {
        &self.limit
    }

    fn mut_limit_for_reflect(&mut self) -> &mut u64 {
        &mut self.limit
    }
}

impl ::protobuf::Message for ScanMvccRequest {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.from_key)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.to_key)?;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.limit = tmp;
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
        if !self.from_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.from_key);
        }
        if !self.to_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.to_key);
        }
        if self.limit != 0 {
            my_size += ::protobuf::rt::value_size(3, self.limit, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.from_key.is_empty() {
            os.write_bytes(1, &self.from_key)?;
        }
        if !self.to_key.is_empty() {
            os.write_bytes(2, &self.to_key)?;
        }
        if self.limit != 0 {
            os.write_uint64(3, self.limit)?;
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

impl ::protobuf::MessageStatic for ScanMvccRequest {
    fn new() -> ScanMvccRequest {
        ScanMvccRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<ScanMvccRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "from_key",
                    ScanMvccRequest::get_from_key_for_reflect,
                    ScanMvccRequest::mut_from_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "to_key",
                    ScanMvccRequest::get_to_key_for_reflect,
                    ScanMvccRequest::mut_to_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "limit",
                    ScanMvccRequest::get_limit_for_reflect,
                    ScanMvccRequest::mut_limit_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ScanMvccRequest>(
                    "ScanMvccRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ScanMvccRequest {
    fn clear(&mut self) {
        self.clear_from_key();
        self.clear_to_key();
        self.clear_limit();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ScanMvccRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ScanMvccRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ScanMvccResponse {
    // message fields
    pub key: ::std::vec::Vec<u8>,
    pub info: ::protobuf::SingularPtrField<super::kvrpcpb::MvccInfo>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ScanMvccResponse {}

impl ScanMvccResponse {
    pub fn new() -> ScanMvccResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ScanMvccResponse {
        static mut instance: ::protobuf::lazy::Lazy<ScanMvccResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ScanMvccResponse,
        };
        unsafe {
            instance.get(ScanMvccResponse::new)
        }
    }

    // bytes key = 1;

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

    // .kvrpcpb.MvccInfo info = 2;

    pub fn clear_info(&mut self) {
        self.info.clear();
    }

    pub fn has_info(&self) -> bool {
        self.info.is_some()
    }

    // Param is passed by value, moved
    pub fn set_info(&mut self, v: super::kvrpcpb::MvccInfo) {
        self.info = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_info(&mut self) -> &mut super::kvrpcpb::MvccInfo {
        if self.info.is_none() {
            self.info.set_default();
        }
        self.info.as_mut().unwrap()
    }

    // Take field
    pub fn take_info(&mut self) -> super::kvrpcpb::MvccInfo {
        self.info.take().unwrap_or_else(|| super::kvrpcpb::MvccInfo::new())
    }

    pub fn get_info(&self) -> &super::kvrpcpb::MvccInfo {
        self.info.as_ref().unwrap_or_else(|| super::kvrpcpb::MvccInfo::default_instance())
    }

    fn get_info_for_reflect(&self) -> &::protobuf::SingularPtrField<super::kvrpcpb::MvccInfo> {
        &self.info
    }

    fn mut_info_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::kvrpcpb::MvccInfo> {
        &mut self.info
    }
}

impl ::protobuf::Message for ScanMvccResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.info {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.info)?;
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
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.key);
        }
        if let Some(ref v) = self.info.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.key.is_empty() {
            os.write_bytes(1, &self.key)?;
        }
        if let Some(ref v) = self.info.as_ref() {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for ScanMvccResponse {
    fn new() -> ScanMvccResponse {
        ScanMvccResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<ScanMvccResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    ScanMvccResponse::get_key_for_reflect,
                    ScanMvccResponse::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::kvrpcpb::MvccInfo>>(
                    "info",
                    ScanMvccResponse::get_info_for_reflect,
                    ScanMvccResponse::mut_info_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ScanMvccResponse>(
                    "ScanMvccResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ScanMvccResponse {
    fn clear(&mut self) {
        self.clear_key();
        self.clear_info();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ScanMvccResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ScanMvccResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct CompactRequest {
    // message fields
    pub db: DB,
    pub cf: ::std::string::String,
    pub from_key: ::std::vec::Vec<u8>,
    pub to_key: ::std::vec::Vec<u8>,
    pub threads: u32,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for CompactRequest {}

impl CompactRequest {
    pub fn new() -> CompactRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static CompactRequest {
        static mut instance: ::protobuf::lazy::Lazy<CompactRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const CompactRequest,
        };
        unsafe {
            instance.get(CompactRequest::new)
        }
    }

    // .debugpb.DB db = 1;

    pub fn clear_db(&mut self) {
        self.db = DB::INVALID;
    }

    // Param is passed by value, moved
    pub fn set_db(&mut self, v: DB) {
        self.db = v;
    }

    pub fn get_db(&self) -> DB {
        self.db
    }

    fn get_db_for_reflect(&self) -> &DB {
        &self.db
    }

    fn mut_db_for_reflect(&mut self) -> &mut DB {
        &mut self.db
    }

    // string cf = 2;

    pub fn clear_cf(&mut self) {
        self.cf.clear();
    }

    // Param is passed by value, moved
    pub fn set_cf(&mut self, v: ::std::string::String) {
        self.cf = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_cf(&mut self) -> &mut ::std::string::String {
        &mut self.cf
    }

    // Take field
    pub fn take_cf(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.cf, ::std::string::String::new())
    }

    pub fn get_cf(&self) -> &str {
        &self.cf
    }

    fn get_cf_for_reflect(&self) -> &::std::string::String {
        &self.cf
    }

    fn mut_cf_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.cf
    }

    // bytes from_key = 3;

    pub fn clear_from_key(&mut self) {
        self.from_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_from_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.from_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_from_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.from_key
    }

    // Take field
    pub fn take_from_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.from_key, ::std::vec::Vec::new())
    }

    pub fn get_from_key(&self) -> &[u8] {
        &self.from_key
    }

    fn get_from_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.from_key
    }

    fn mut_from_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.from_key
    }

    // bytes to_key = 4;

    pub fn clear_to_key(&mut self) {
        self.to_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_to_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.to_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_to_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.to_key
    }

    // Take field
    pub fn take_to_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.to_key, ::std::vec::Vec::new())
    }

    pub fn get_to_key(&self) -> &[u8] {
        &self.to_key
    }

    fn get_to_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.to_key
    }

    fn mut_to_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.to_key
    }

    // uint32 threads = 5;

    pub fn clear_threads(&mut self) {
        self.threads = 0;
    }

    // Param is passed by value, moved
    pub fn set_threads(&mut self, v: u32) {
        self.threads = v;
    }

    pub fn get_threads(&self) -> u32 {
        self.threads
    }

    fn get_threads_for_reflect(&self) -> &u32 {
        &self.threads
    }

    fn mut_threads_for_reflect(&mut self) -> &mut u32 {
        &mut self.threads
    }
}

impl ::protobuf::Message for CompactRequest {
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
                    self.db = tmp;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.from_key)?;
                },
                4 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.to_key)?;
                },
                5 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint32()?;
                    self.threads = tmp;
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
        if self.db != DB::INVALID {
            my_size += ::protobuf::rt::enum_size(1, self.db);
        }
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.cf);
        }
        if !self.from_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.from_key);
        }
        if !self.to_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(4, &self.to_key);
        }
        if self.threads != 0 {
            my_size += ::protobuf::rt::value_size(5, self.threads, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.db != DB::INVALID {
            os.write_enum(1, self.db.value())?;
        }
        if !self.cf.is_empty() {
            os.write_string(2, &self.cf)?;
        }
        if !self.from_key.is_empty() {
            os.write_bytes(3, &self.from_key)?;
        }
        if !self.to_key.is_empty() {
            os.write_bytes(4, &self.to_key)?;
        }
        if self.threads != 0 {
            os.write_uint32(5, self.threads)?;
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

impl ::protobuf::MessageStatic for CompactRequest {
    fn new() -> CompactRequest {
        CompactRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<CompactRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeEnum<DB>>(
                    "db",
                    CompactRequest::get_db_for_reflect,
                    CompactRequest::mut_db_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    CompactRequest::get_cf_for_reflect,
                    CompactRequest::mut_cf_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "from_key",
                    CompactRequest::get_from_key_for_reflect,
                    CompactRequest::mut_from_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "to_key",
                    CompactRequest::get_to_key_for_reflect,
                    CompactRequest::mut_to_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint32>(
                    "threads",
                    CompactRequest::get_threads_for_reflect,
                    CompactRequest::mut_threads_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<CompactRequest>(
                    "CompactRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for CompactRequest {
    fn clear(&mut self) {
        self.clear_db();
        self.clear_cf();
        self.clear_from_key();
        self.clear_to_key();
        self.clear_threads();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for CompactRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for CompactRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct CompactResponse {
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for CompactResponse {}

impl CompactResponse {
    pub fn new() -> CompactResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static CompactResponse {
        static mut instance: ::protobuf::lazy::Lazy<CompactResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const CompactResponse,
        };
        unsafe {
            instance.get(CompactResponse::new)
        }
    }
}

impl ::protobuf::Message for CompactResponse {
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

impl ::protobuf::MessageStatic for CompactResponse {
    fn new() -> CompactResponse {
        CompactResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<CompactResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let fields = ::std::vec::Vec::new();
                ::protobuf::reflect::MessageDescriptor::new::<CompactResponse>(
                    "CompactResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for CompactResponse {
    fn clear(&mut self) {
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for CompactResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for CompactResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct InjectFailPointRequest {
    // message fields
    pub name: ::std::string::String,
    pub actions: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for InjectFailPointRequest {}

impl InjectFailPointRequest {
    pub fn new() -> InjectFailPointRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static InjectFailPointRequest {
        static mut instance: ::protobuf::lazy::Lazy<InjectFailPointRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const InjectFailPointRequest,
        };
        unsafe {
            instance.get(InjectFailPointRequest::new)
        }
    }

    // string name = 1;

    pub fn clear_name(&mut self) {
        self.name.clear();
    }

    // Param is passed by value, moved
    pub fn set_name(&mut self, v: ::std::string::String) {
        self.name = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_name(&mut self) -> &mut ::std::string::String {
        &mut self.name
    }

    // Take field
    pub fn take_name(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.name, ::std::string::String::new())
    }

    pub fn get_name(&self) -> &str {
        &self.name
    }

    fn get_name_for_reflect(&self) -> &::std::string::String {
        &self.name
    }

    fn mut_name_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.name
    }

    // string actions = 2;

    pub fn clear_actions(&mut self) {
        self.actions.clear();
    }

    // Param is passed by value, moved
    pub fn set_actions(&mut self, v: ::std::string::String) {
        self.actions = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_actions(&mut self) -> &mut ::std::string::String {
        &mut self.actions
    }

    // Take field
    pub fn take_actions(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.actions, ::std::string::String::new())
    }

    pub fn get_actions(&self) -> &str {
        &self.actions
    }

    fn get_actions_for_reflect(&self) -> &::std::string::String {
        &self.actions
    }

    fn mut_actions_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.actions
    }
}

impl ::protobuf::Message for InjectFailPointRequest {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.name)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.actions)?;
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
        if !self.name.is_empty() {
            my_size += ::protobuf::rt::string_size(1, &self.name);
        }
        if !self.actions.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.actions);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.name.is_empty() {
            os.write_string(1, &self.name)?;
        }
        if !self.actions.is_empty() {
            os.write_string(2, &self.actions)?;
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

impl ::protobuf::MessageStatic for InjectFailPointRequest {
    fn new() -> InjectFailPointRequest {
        InjectFailPointRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<InjectFailPointRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "name",
                    InjectFailPointRequest::get_name_for_reflect,
                    InjectFailPointRequest::mut_name_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "actions",
                    InjectFailPointRequest::get_actions_for_reflect,
                    InjectFailPointRequest::mut_actions_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<InjectFailPointRequest>(
                    "InjectFailPointRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for InjectFailPointRequest {
    fn clear(&mut self) {
        self.clear_name();
        self.clear_actions();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for InjectFailPointRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for InjectFailPointRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct InjectFailPointResponse {
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for InjectFailPointResponse {}

impl InjectFailPointResponse {
    pub fn new() -> InjectFailPointResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static InjectFailPointResponse {
        static mut instance: ::protobuf::lazy::Lazy<InjectFailPointResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const InjectFailPointResponse,
        };
        unsafe {
            instance.get(InjectFailPointResponse::new)
        }
    }
}

impl ::protobuf::Message for InjectFailPointResponse {
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

impl ::protobuf::MessageStatic for InjectFailPointResponse {
    fn new() -> InjectFailPointResponse {
        InjectFailPointResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<InjectFailPointResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let fields = ::std::vec::Vec::new();
                ::protobuf::reflect::MessageDescriptor::new::<InjectFailPointResponse>(
                    "InjectFailPointResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for InjectFailPointResponse {
    fn clear(&mut self) {
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for InjectFailPointResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for InjectFailPointResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RecoverFailPointRequest {
    // message fields
    pub name: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RecoverFailPointRequest {}

impl RecoverFailPointRequest {
    pub fn new() -> RecoverFailPointRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RecoverFailPointRequest {
        static mut instance: ::protobuf::lazy::Lazy<RecoverFailPointRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RecoverFailPointRequest,
        };
        unsafe {
            instance.get(RecoverFailPointRequest::new)
        }
    }

    // string name = 1;

    pub fn clear_name(&mut self) {
        self.name.clear();
    }

    // Param is passed by value, moved
    pub fn set_name(&mut self, v: ::std::string::String) {
        self.name = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_name(&mut self) -> &mut ::std::string::String {
        &mut self.name
    }

    // Take field
    pub fn take_name(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.name, ::std::string::String::new())
    }

    pub fn get_name(&self) -> &str {
        &self.name
    }

    fn get_name_for_reflect(&self) -> &::std::string::String {
        &self.name
    }

    fn mut_name_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.name
    }
}

impl ::protobuf::Message for RecoverFailPointRequest {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.name)?;
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
        if !self.name.is_empty() {
            my_size += ::protobuf::rt::string_size(1, &self.name);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.name.is_empty() {
            os.write_string(1, &self.name)?;
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

impl ::protobuf::MessageStatic for RecoverFailPointRequest {
    fn new() -> RecoverFailPointRequest {
        RecoverFailPointRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RecoverFailPointRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "name",
                    RecoverFailPointRequest::get_name_for_reflect,
                    RecoverFailPointRequest::mut_name_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RecoverFailPointRequest>(
                    "RecoverFailPointRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RecoverFailPointRequest {
    fn clear(&mut self) {
        self.clear_name();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RecoverFailPointRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RecoverFailPointRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RecoverFailPointResponse {
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RecoverFailPointResponse {}

impl RecoverFailPointResponse {
    pub fn new() -> RecoverFailPointResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RecoverFailPointResponse {
        static mut instance: ::protobuf::lazy::Lazy<RecoverFailPointResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RecoverFailPointResponse,
        };
        unsafe {
            instance.get(RecoverFailPointResponse::new)
        }
    }
}

impl ::protobuf::Message for RecoverFailPointResponse {
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

impl ::protobuf::MessageStatic for RecoverFailPointResponse {
    fn new() -> RecoverFailPointResponse {
        RecoverFailPointResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RecoverFailPointResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let fields = ::std::vec::Vec::new();
                ::protobuf::reflect::MessageDescriptor::new::<RecoverFailPointResponse>(
                    "RecoverFailPointResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RecoverFailPointResponse {
    fn clear(&mut self) {
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RecoverFailPointResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RecoverFailPointResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ListFailPointsRequest {
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ListFailPointsRequest {}

impl ListFailPointsRequest {
    pub fn new() -> ListFailPointsRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ListFailPointsRequest {
        static mut instance: ::protobuf::lazy::Lazy<ListFailPointsRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ListFailPointsRequest,
        };
        unsafe {
            instance.get(ListFailPointsRequest::new)
        }
    }
}

impl ::protobuf::Message for ListFailPointsRequest {
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

impl ::protobuf::MessageStatic for ListFailPointsRequest {
    fn new() -> ListFailPointsRequest {
        ListFailPointsRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<ListFailPointsRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let fields = ::std::vec::Vec::new();
                ::protobuf::reflect::MessageDescriptor::new::<ListFailPointsRequest>(
                    "ListFailPointsRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ListFailPointsRequest {
    fn clear(&mut self) {
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ListFailPointsRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ListFailPointsRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ListFailPointsResponse {
    // message fields
    pub entries: ::protobuf::RepeatedField<ListFailPointsResponse_Entry>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ListFailPointsResponse {}

impl ListFailPointsResponse {
    pub fn new() -> ListFailPointsResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ListFailPointsResponse {
        static mut instance: ::protobuf::lazy::Lazy<ListFailPointsResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ListFailPointsResponse,
        };
        unsafe {
            instance.get(ListFailPointsResponse::new)
        }
    }

    // repeated .debugpb.ListFailPointsResponse.Entry entries = 1;

    pub fn clear_entries(&mut self) {
        self.entries.clear();
    }

    // Param is passed by value, moved
    pub fn set_entries(&mut self, v: ::protobuf::RepeatedField<ListFailPointsResponse_Entry>) {
        self.entries = v;
    }

    // Mutable pointer to the field.
    pub fn mut_entries(&mut self) -> &mut ::protobuf::RepeatedField<ListFailPointsResponse_Entry> {
        &mut self.entries
    }

    // Take field
    pub fn take_entries(&mut self) -> ::protobuf::RepeatedField<ListFailPointsResponse_Entry> {
        ::std::mem::replace(&mut self.entries, ::protobuf::RepeatedField::new())
    }

    pub fn get_entries(&self) -> &[ListFailPointsResponse_Entry] {
        &self.entries
    }

    fn get_entries_for_reflect(&self) -> &::protobuf::RepeatedField<ListFailPointsResponse_Entry> {
        &self.entries
    }

    fn mut_entries_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<ListFailPointsResponse_Entry> {
        &mut self.entries
    }
}

impl ::protobuf::Message for ListFailPointsResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.entries {
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
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.entries)?;
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
        for value in &self.entries {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        for v in &self.entries {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for ListFailPointsResponse {
    fn new() -> ListFailPointsResponse {
        ListFailPointsResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<ListFailPointsResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<ListFailPointsResponse_Entry>>(
                    "entries",
                    ListFailPointsResponse::get_entries_for_reflect,
                    ListFailPointsResponse::mut_entries_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ListFailPointsResponse>(
                    "ListFailPointsResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ListFailPointsResponse {
    fn clear(&mut self) {
        self.clear_entries();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ListFailPointsResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ListFailPointsResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ListFailPointsResponse_Entry {
    // message fields
    pub name: ::std::string::String,
    pub actions: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ListFailPointsResponse_Entry {}

impl ListFailPointsResponse_Entry {
    pub fn new() -> ListFailPointsResponse_Entry {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ListFailPointsResponse_Entry {
        static mut instance: ::protobuf::lazy::Lazy<ListFailPointsResponse_Entry> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ListFailPointsResponse_Entry,
        };
        unsafe {
            instance.get(ListFailPointsResponse_Entry::new)
        }
    }

    // string name = 1;

    pub fn clear_name(&mut self) {
        self.name.clear();
    }

    // Param is passed by value, moved
    pub fn set_name(&mut self, v: ::std::string::String) {
        self.name = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_name(&mut self) -> &mut ::std::string::String {
        &mut self.name
    }

    // Take field
    pub fn take_name(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.name, ::std::string::String::new())
    }

    pub fn get_name(&self) -> &str {
        &self.name
    }

    fn get_name_for_reflect(&self) -> &::std::string::String {
        &self.name
    }

    fn mut_name_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.name
    }

    // string actions = 2;

    pub fn clear_actions(&mut self) {
        self.actions.clear();
    }

    // Param is passed by value, moved
    pub fn set_actions(&mut self, v: ::std::string::String) {
        self.actions = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_actions(&mut self) -> &mut ::std::string::String {
        &mut self.actions
    }

    // Take field
    pub fn take_actions(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.actions, ::std::string::String::new())
    }

    pub fn get_actions(&self) -> &str {
        &self.actions
    }

    fn get_actions_for_reflect(&self) -> &::std::string::String {
        &self.actions
    }

    fn mut_actions_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.actions
    }
}

impl ::protobuf::Message for ListFailPointsResponse_Entry {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.name)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.actions)?;
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
        if !self.name.is_empty() {
            my_size += ::protobuf::rt::string_size(1, &self.name);
        }
        if !self.actions.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.actions);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.name.is_empty() {
            os.write_string(1, &self.name)?;
        }
        if !self.actions.is_empty() {
            os.write_string(2, &self.actions)?;
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

impl ::protobuf::MessageStatic for ListFailPointsResponse_Entry {
    fn new() -> ListFailPointsResponse_Entry {
        ListFailPointsResponse_Entry::new()
    }

    fn descriptor_static(_: ::std::option::Option<ListFailPointsResponse_Entry>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "name",
                    ListFailPointsResponse_Entry::get_name_for_reflect,
                    ListFailPointsResponse_Entry::mut_name_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "actions",
                    ListFailPointsResponse_Entry::get_actions_for_reflect,
                    ListFailPointsResponse_Entry::mut_actions_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ListFailPointsResponse_Entry>(
                    "ListFailPointsResponse_Entry",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ListFailPointsResponse_Entry {
    fn clear(&mut self) {
        self.clear_name();
        self.clear_actions();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ListFailPointsResponse_Entry {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ListFailPointsResponse_Entry {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct GetMetricsRequest {
    // message fields
    pub all: bool,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for GetMetricsRequest {}

impl GetMetricsRequest {
    pub fn new() -> GetMetricsRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static GetMetricsRequest {
        static mut instance: ::protobuf::lazy::Lazy<GetMetricsRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const GetMetricsRequest,
        };
        unsafe {
            instance.get(GetMetricsRequest::new)
        }
    }

    // bool all = 1;

    pub fn clear_all(&mut self) {
        self.all = false;
    }

    // Param is passed by value, moved
    pub fn set_all(&mut self, v: bool) {
        self.all = v;
    }

    pub fn get_all(&self) -> bool {
        self.all
    }

    fn get_all_for_reflect(&self) -> &bool {
        &self.all
    }

    fn mut_all_for_reflect(&mut self) -> &mut bool {
        &mut self.all
    }
}

impl ::protobuf::Message for GetMetricsRequest {
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
                    let tmp = is.read_bool()?;
                    self.all = tmp;
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
        if self.all != false {
            my_size += 2;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.all != false {
            os.write_bool(1, self.all)?;
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

impl ::protobuf::MessageStatic for GetMetricsRequest {
    fn new() -> GetMetricsRequest {
        GetMetricsRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<GetMetricsRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "all",
                    GetMetricsRequest::get_all_for_reflect,
                    GetMetricsRequest::mut_all_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<GetMetricsRequest>(
                    "GetMetricsRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for GetMetricsRequest {
    fn clear(&mut self) {
        self.clear_all();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for GetMetricsRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for GetMetricsRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct GetMetricsResponse {
    // message fields
    pub prometheus: ::std::string::String,
    pub rocksdb_kv: ::std::string::String,
    pub rocksdb_raft: ::std::string::String,
    pub jemalloc: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for GetMetricsResponse {}

impl GetMetricsResponse {
    pub fn new() -> GetMetricsResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static GetMetricsResponse {
        static mut instance: ::protobuf::lazy::Lazy<GetMetricsResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const GetMetricsResponse,
        };
        unsafe {
            instance.get(GetMetricsResponse::new)
        }
    }

    // string prometheus = 1;

    pub fn clear_prometheus(&mut self) {
        self.prometheus.clear();
    }

    // Param is passed by value, moved
    pub fn set_prometheus(&mut self, v: ::std::string::String) {
        self.prometheus = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_prometheus(&mut self) -> &mut ::std::string::String {
        &mut self.prometheus
    }

    // Take field
    pub fn take_prometheus(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.prometheus, ::std::string::String::new())
    }

    pub fn get_prometheus(&self) -> &str {
        &self.prometheus
    }

    fn get_prometheus_for_reflect(&self) -> &::std::string::String {
        &self.prometheus
    }

    fn mut_prometheus_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.prometheus
    }

    // string rocksdb_kv = 2;

    pub fn clear_rocksdb_kv(&mut self) {
        self.rocksdb_kv.clear();
    }

    // Param is passed by value, moved
    pub fn set_rocksdb_kv(&mut self, v: ::std::string::String) {
        self.rocksdb_kv = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_rocksdb_kv(&mut self) -> &mut ::std::string::String {
        &mut self.rocksdb_kv
    }

    // Take field
    pub fn take_rocksdb_kv(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.rocksdb_kv, ::std::string::String::new())
    }

    pub fn get_rocksdb_kv(&self) -> &str {
        &self.rocksdb_kv
    }

    fn get_rocksdb_kv_for_reflect(&self) -> &::std::string::String {
        &self.rocksdb_kv
    }

    fn mut_rocksdb_kv_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.rocksdb_kv
    }

    // string rocksdb_raft = 3;

    pub fn clear_rocksdb_raft(&mut self) {
        self.rocksdb_raft.clear();
    }

    // Param is passed by value, moved
    pub fn set_rocksdb_raft(&mut self, v: ::std::string::String) {
        self.rocksdb_raft = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_rocksdb_raft(&mut self) -> &mut ::std::string::String {
        &mut self.rocksdb_raft
    }

    // Take field
    pub fn take_rocksdb_raft(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.rocksdb_raft, ::std::string::String::new())
    }

    pub fn get_rocksdb_raft(&self) -> &str {
        &self.rocksdb_raft
    }

    fn get_rocksdb_raft_for_reflect(&self) -> &::std::string::String {
        &self.rocksdb_raft
    }

    fn mut_rocksdb_raft_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.rocksdb_raft
    }

    // string jemalloc = 4;

    pub fn clear_jemalloc(&mut self) {
        self.jemalloc.clear();
    }

    // Param is passed by value, moved
    pub fn set_jemalloc(&mut self, v: ::std::string::String) {
        self.jemalloc = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_jemalloc(&mut self) -> &mut ::std::string::String {
        &mut self.jemalloc
    }

    // Take field
    pub fn take_jemalloc(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.jemalloc, ::std::string::String::new())
    }

    pub fn get_jemalloc(&self) -> &str {
        &self.jemalloc
    }

    fn get_jemalloc_for_reflect(&self) -> &::std::string::String {
        &self.jemalloc
    }

    fn mut_jemalloc_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.jemalloc
    }
}

impl ::protobuf::Message for GetMetricsResponse {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.prometheus)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.rocksdb_kv)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.rocksdb_raft)?;
                },
                4 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.jemalloc)?;
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
        if !self.prometheus.is_empty() {
            my_size += ::protobuf::rt::string_size(1, &self.prometheus);
        }
        if !self.rocksdb_kv.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.rocksdb_kv);
        }
        if !self.rocksdb_raft.is_empty() {
            my_size += ::protobuf::rt::string_size(3, &self.rocksdb_raft);
        }
        if !self.jemalloc.is_empty() {
            my_size += ::protobuf::rt::string_size(4, &self.jemalloc);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.prometheus.is_empty() {
            os.write_string(1, &self.prometheus)?;
        }
        if !self.rocksdb_kv.is_empty() {
            os.write_string(2, &self.rocksdb_kv)?;
        }
        if !self.rocksdb_raft.is_empty() {
            os.write_string(3, &self.rocksdb_raft)?;
        }
        if !self.jemalloc.is_empty() {
            os.write_string(4, &self.jemalloc)?;
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

impl ::protobuf::MessageStatic for GetMetricsResponse {
    fn new() -> GetMetricsResponse {
        GetMetricsResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<GetMetricsResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "prometheus",
                    GetMetricsResponse::get_prometheus_for_reflect,
                    GetMetricsResponse::mut_prometheus_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "rocksdb_kv",
                    GetMetricsResponse::get_rocksdb_kv_for_reflect,
                    GetMetricsResponse::mut_rocksdb_kv_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "rocksdb_raft",
                    GetMetricsResponse::get_rocksdb_raft_for_reflect,
                    GetMetricsResponse::mut_rocksdb_raft_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "jemalloc",
                    GetMetricsResponse::get_jemalloc_for_reflect,
                    GetMetricsResponse::mut_jemalloc_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<GetMetricsResponse>(
                    "GetMetricsResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for GetMetricsResponse {
    fn clear(&mut self) {
        self.clear_prometheus();
        self.clear_rocksdb_kv();
        self.clear_rocksdb_raft();
        self.clear_jemalloc();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for GetMetricsResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for GetMetricsResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RegionConsistencyCheckRequest {
    // message fields
    pub region_id: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RegionConsistencyCheckRequest {}

impl RegionConsistencyCheckRequest {
    pub fn new() -> RegionConsistencyCheckRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RegionConsistencyCheckRequest {
        static mut instance: ::protobuf::lazy::Lazy<RegionConsistencyCheckRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RegionConsistencyCheckRequest,
        };
        unsafe {
            instance.get(RegionConsistencyCheckRequest::new)
        }
    }

    // uint64 region_id = 1;

    pub fn clear_region_id(&mut self) {
        self.region_id = 0;
    }

    // Param is passed by value, moved
    pub fn set_region_id(&mut self, v: u64) {
        self.region_id = v;
    }

    pub fn get_region_id(&self) -> u64 {
        self.region_id
    }

    fn get_region_id_for_reflect(&self) -> &u64 {
        &self.region_id
    }

    fn mut_region_id_for_reflect(&mut self) -> &mut u64 {
        &mut self.region_id
    }
}

impl ::protobuf::Message for RegionConsistencyCheckRequest {
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
                    let tmp = is.read_uint64()?;
                    self.region_id = tmp;
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
        if self.region_id != 0 {
            my_size += ::protobuf::rt::value_size(1, self.region_id, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.region_id != 0 {
            os.write_uint64(1, self.region_id)?;
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

impl ::protobuf::MessageStatic for RegionConsistencyCheckRequest {
    fn new() -> RegionConsistencyCheckRequest {
        RegionConsistencyCheckRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RegionConsistencyCheckRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "region_id",
                    RegionConsistencyCheckRequest::get_region_id_for_reflect,
                    RegionConsistencyCheckRequest::mut_region_id_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RegionConsistencyCheckRequest>(
                    "RegionConsistencyCheckRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RegionConsistencyCheckRequest {
    fn clear(&mut self) {
        self.clear_region_id();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RegionConsistencyCheckRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RegionConsistencyCheckRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RegionConsistencyCheckResponse {
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RegionConsistencyCheckResponse {}

impl RegionConsistencyCheckResponse {
    pub fn new() -> RegionConsistencyCheckResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RegionConsistencyCheckResponse {
        static mut instance: ::protobuf::lazy::Lazy<RegionConsistencyCheckResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RegionConsistencyCheckResponse,
        };
        unsafe {
            instance.get(RegionConsistencyCheckResponse::new)
        }
    }
}

impl ::protobuf::Message for RegionConsistencyCheckResponse {
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

impl ::protobuf::MessageStatic for RegionConsistencyCheckResponse {
    fn new() -> RegionConsistencyCheckResponse {
        RegionConsistencyCheckResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RegionConsistencyCheckResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let fields = ::std::vec::Vec::new();
                ::protobuf::reflect::MessageDescriptor::new::<RegionConsistencyCheckResponse>(
                    "RegionConsistencyCheckResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RegionConsistencyCheckResponse {
    fn clear(&mut self) {
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RegionConsistencyCheckResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RegionConsistencyCheckResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ModifyTikvConfigRequest {
    // message fields
    pub module: MODULE,
    pub config_name: ::std::string::String,
    pub config_value: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ModifyTikvConfigRequest {}

impl ModifyTikvConfigRequest {
    pub fn new() -> ModifyTikvConfigRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ModifyTikvConfigRequest {
        static mut instance: ::protobuf::lazy::Lazy<ModifyTikvConfigRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ModifyTikvConfigRequest,
        };
        unsafe {
            instance.get(ModifyTikvConfigRequest::new)
        }
    }

    // .debugpb.MODULE module = 1;

    pub fn clear_module(&mut self) {
        self.module = MODULE::UNUSED;
    }

    // Param is passed by value, moved
    pub fn set_module(&mut self, v: MODULE) {
        self.module = v;
    }

    pub fn get_module(&self) -> MODULE {
        self.module
    }

    fn get_module_for_reflect(&self) -> &MODULE {
        &self.module
    }

    fn mut_module_for_reflect(&mut self) -> &mut MODULE {
        &mut self.module
    }

    // string config_name = 2;

    pub fn clear_config_name(&mut self) {
        self.config_name.clear();
    }

    // Param is passed by value, moved
    pub fn set_config_name(&mut self, v: ::std::string::String) {
        self.config_name = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_config_name(&mut self) -> &mut ::std::string::String {
        &mut self.config_name
    }

    // Take field
    pub fn take_config_name(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.config_name, ::std::string::String::new())
    }

    pub fn get_config_name(&self) -> &str {
        &self.config_name
    }

    fn get_config_name_for_reflect(&self) -> &::std::string::String {
        &self.config_name
    }

    fn mut_config_name_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.config_name
    }

    // string config_value = 3;

    pub fn clear_config_value(&mut self) {
        self.config_value.clear();
    }

    // Param is passed by value, moved
    pub fn set_config_value(&mut self, v: ::std::string::String) {
        self.config_value = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_config_value(&mut self) -> &mut ::std::string::String {
        &mut self.config_value
    }

    // Take field
    pub fn take_config_value(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.config_value, ::std::string::String::new())
    }

    pub fn get_config_value(&self) -> &str {
        &self.config_value
    }

    fn get_config_value_for_reflect(&self) -> &::std::string::String {
        &self.config_value
    }

    fn mut_config_value_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.config_value
    }
}

impl ::protobuf::Message for ModifyTikvConfigRequest {
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
                    self.module = tmp;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.config_name)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.config_value)?;
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
        if self.module != MODULE::UNUSED {
            my_size += ::protobuf::rt::enum_size(1, self.module);
        }
        if !self.config_name.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.config_name);
        }
        if !self.config_value.is_empty() {
            my_size += ::protobuf::rt::string_size(3, &self.config_value);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.module != MODULE::UNUSED {
            os.write_enum(1, self.module.value())?;
        }
        if !self.config_name.is_empty() {
            os.write_string(2, &self.config_name)?;
        }
        if !self.config_value.is_empty() {
            os.write_string(3, &self.config_value)?;
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

impl ::protobuf::MessageStatic for ModifyTikvConfigRequest {
    fn new() -> ModifyTikvConfigRequest {
        ModifyTikvConfigRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<ModifyTikvConfigRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeEnum<MODULE>>(
                    "module",
                    ModifyTikvConfigRequest::get_module_for_reflect,
                    ModifyTikvConfigRequest::mut_module_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "config_name",
                    ModifyTikvConfigRequest::get_config_name_for_reflect,
                    ModifyTikvConfigRequest::mut_config_name_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "config_value",
                    ModifyTikvConfigRequest::get_config_value_for_reflect,
                    ModifyTikvConfigRequest::mut_config_value_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ModifyTikvConfigRequest>(
                    "ModifyTikvConfigRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ModifyTikvConfigRequest {
    fn clear(&mut self) {
        self.clear_module();
        self.clear_config_name();
        self.clear_config_value();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ModifyTikvConfigRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ModifyTikvConfigRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ModifyTikvConfigResponse {
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ModifyTikvConfigResponse {}

impl ModifyTikvConfigResponse {
    pub fn new() -> ModifyTikvConfigResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ModifyTikvConfigResponse {
        static mut instance: ::protobuf::lazy::Lazy<ModifyTikvConfigResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ModifyTikvConfigResponse,
        };
        unsafe {
            instance.get(ModifyTikvConfigResponse::new)
        }
    }
}

impl ::protobuf::Message for ModifyTikvConfigResponse {
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

impl ::protobuf::MessageStatic for ModifyTikvConfigResponse {
    fn new() -> ModifyTikvConfigResponse {
        ModifyTikvConfigResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<ModifyTikvConfigResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let fields = ::std::vec::Vec::new();
                ::protobuf::reflect::MessageDescriptor::new::<ModifyTikvConfigResponse>(
                    "ModifyTikvConfigResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ModifyTikvConfigResponse {
    fn clear(&mut self) {
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ModifyTikvConfigResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ModifyTikvConfigResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct Property {
    // message fields
    pub name: ::std::string::String,
    pub value: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for Property {}

impl Property {
    pub fn new() -> Property {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static Property {
        static mut instance: ::protobuf::lazy::Lazy<Property> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const Property,
        };
        unsafe {
            instance.get(Property::new)
        }
    }

    // string name = 1;

    pub fn clear_name(&mut self) {
        self.name.clear();
    }

    // Param is passed by value, moved
    pub fn set_name(&mut self, v: ::std::string::String) {
        self.name = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_name(&mut self) -> &mut ::std::string::String {
        &mut self.name
    }

    // Take field
    pub fn take_name(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.name, ::std::string::String::new())
    }

    pub fn get_name(&self) -> &str {
        &self.name
    }

    fn get_name_for_reflect(&self) -> &::std::string::String {
        &self.name
    }

    fn mut_name_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.name
    }

    // string value = 2;

    pub fn clear_value(&mut self) {
        self.value.clear();
    }

    // Param is passed by value, moved
    pub fn set_value(&mut self, v: ::std::string::String) {
        self.value = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_value(&mut self) -> &mut ::std::string::String {
        &mut self.value
    }

    // Take field
    pub fn take_value(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.value, ::std::string::String::new())
    }

    pub fn get_value(&self) -> &str {
        &self.value
    }

    fn get_value_for_reflect(&self) -> &::std::string::String {
        &self.value
    }

    fn mut_value_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.value
    }
}

impl ::protobuf::Message for Property {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.name)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.value)?;
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
        if !self.name.is_empty() {
            my_size += ::protobuf::rt::string_size(1, &self.name);
        }
        if !self.value.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.value);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.name.is_empty() {
            os.write_string(1, &self.name)?;
        }
        if !self.value.is_empty() {
            os.write_string(2, &self.value)?;
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

impl ::protobuf::MessageStatic for Property {
    fn new() -> Property {
        Property::new()
    }

    fn descriptor_static(_: ::std::option::Option<Property>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "name",
                    Property::get_name_for_reflect,
                    Property::mut_name_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "value",
                    Property::get_value_for_reflect,
                    Property::mut_value_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<Property>(
                    "Property",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for Property {
    fn clear(&mut self) {
        self.clear_name();
        self.clear_value();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for Property {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for Property {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct GetRegionPropertiesRequest {
    // message fields
    pub region_id: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for GetRegionPropertiesRequest {}

impl GetRegionPropertiesRequest {
    pub fn new() -> GetRegionPropertiesRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static GetRegionPropertiesRequest {
        static mut instance: ::protobuf::lazy::Lazy<GetRegionPropertiesRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const GetRegionPropertiesRequest,
        };
        unsafe {
            instance.get(GetRegionPropertiesRequest::new)
        }
    }

    // uint64 region_id = 1;

    pub fn clear_region_id(&mut self) {
        self.region_id = 0;
    }

    // Param is passed by value, moved
    pub fn set_region_id(&mut self, v: u64) {
        self.region_id = v;
    }

    pub fn get_region_id(&self) -> u64 {
        self.region_id
    }

    fn get_region_id_for_reflect(&self) -> &u64 {
        &self.region_id
    }

    fn mut_region_id_for_reflect(&mut self) -> &mut u64 {
        &mut self.region_id
    }
}

impl ::protobuf::Message for GetRegionPropertiesRequest {
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
                    let tmp = is.read_uint64()?;
                    self.region_id = tmp;
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
        if self.region_id != 0 {
            my_size += ::protobuf::rt::value_size(1, self.region_id, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.region_id != 0 {
            os.write_uint64(1, self.region_id)?;
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

impl ::protobuf::MessageStatic for GetRegionPropertiesRequest {
    fn new() -> GetRegionPropertiesRequest {
        GetRegionPropertiesRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<GetRegionPropertiesRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "region_id",
                    GetRegionPropertiesRequest::get_region_id_for_reflect,
                    GetRegionPropertiesRequest::mut_region_id_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<GetRegionPropertiesRequest>(
                    "GetRegionPropertiesRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for GetRegionPropertiesRequest {
    fn clear(&mut self) {
        self.clear_region_id();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for GetRegionPropertiesRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for GetRegionPropertiesRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct GetRegionPropertiesResponse {
    // message fields
    pub props: ::protobuf::RepeatedField<Property>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for GetRegionPropertiesResponse {}

impl GetRegionPropertiesResponse {
    pub fn new() -> GetRegionPropertiesResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static GetRegionPropertiesResponse {
        static mut instance: ::protobuf::lazy::Lazy<GetRegionPropertiesResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const GetRegionPropertiesResponse,
        };
        unsafe {
            instance.get(GetRegionPropertiesResponse::new)
        }
    }

    // repeated .debugpb.Property props = 1;

    pub fn clear_props(&mut self) {
        self.props.clear();
    }

    // Param is passed by value, moved
    pub fn set_props(&mut self, v: ::protobuf::RepeatedField<Property>) {
        self.props = v;
    }

    // Mutable pointer to the field.
    pub fn mut_props(&mut self) -> &mut ::protobuf::RepeatedField<Property> {
        &mut self.props
    }

    // Take field
    pub fn take_props(&mut self) -> ::protobuf::RepeatedField<Property> {
        ::std::mem::replace(&mut self.props, ::protobuf::RepeatedField::new())
    }

    pub fn get_props(&self) -> &[Property] {
        &self.props
    }

    fn get_props_for_reflect(&self) -> &::protobuf::RepeatedField<Property> {
        &self.props
    }

    fn mut_props_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<Property> {
        &mut self.props
    }
}

impl ::protobuf::Message for GetRegionPropertiesResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.props {
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
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.props)?;
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
        for value in &self.props {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        for v in &self.props {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for GetRegionPropertiesResponse {
    fn new() -> GetRegionPropertiesResponse {
        GetRegionPropertiesResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<GetRegionPropertiesResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Property>>(
                    "props",
                    GetRegionPropertiesResponse::get_props_for_reflect,
                    GetRegionPropertiesResponse::mut_props_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<GetRegionPropertiesResponse>(
                    "GetRegionPropertiesResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for GetRegionPropertiesResponse {
    fn clear(&mut self) {
        self.clear_props();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for GetRegionPropertiesResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for GetRegionPropertiesResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(Clone,PartialEq,Eq,Debug,Hash)]
pub enum DB {
    INVALID = 0,
    KV = 1,
    RAFT = 2,
}

impl ::protobuf::ProtobufEnum for DB {
    fn value(&self) -> i32 {
        *self as i32
    }

    fn from_i32(value: i32) -> ::std::option::Option<DB> {
        match value {
            0 => ::std::option::Option::Some(DB::INVALID),
            1 => ::std::option::Option::Some(DB::KV),
            2 => ::std::option::Option::Some(DB::RAFT),
            _ => ::std::option::Option::None
        }
    }

    fn values() -> &'static [Self] {
        static values: &'static [DB] = &[
            DB::INVALID,
            DB::KV,
            DB::RAFT,
        ];
        values
    }

    fn enum_descriptor_static(_: ::std::option::Option<DB>) -> &'static ::protobuf::reflect::EnumDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::EnumDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::EnumDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                ::protobuf::reflect::EnumDescriptor::new("DB", file_descriptor_proto())
            })
        }
    }
}

impl ::std::marker::Copy for DB {
}

impl ::std::default::Default for DB {
    fn default() -> Self {
        DB::INVALID
    }
}

impl ::protobuf::reflect::ProtobufValue for DB {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Enum(self.descriptor())
    }
}

#[derive(Clone,PartialEq,Eq,Debug,Hash)]
pub enum MODULE {
    UNUSED = 0,
    KVDB = 1,
    RAFTDB = 2,
    READPOOL = 3,
    SERVER = 4,
    STORAGE = 5,
    PD = 6,
    METRIC = 7,
    COPROCESSOR = 8,
    SECURITY = 9,
    IMPORT = 10,
}

impl ::protobuf::ProtobufEnum for MODULE {
    fn value(&self) -> i32 {
        *self as i32
    }

    fn from_i32(value: i32) -> ::std::option::Option<MODULE> {
        match value {
            0 => ::std::option::Option::Some(MODULE::UNUSED),
            1 => ::std::option::Option::Some(MODULE::KVDB),
            2 => ::std::option::Option::Some(MODULE::RAFTDB),
            3 => ::std::option::Option::Some(MODULE::READPOOL),
            4 => ::std::option::Option::Some(MODULE::SERVER),
            5 => ::std::option::Option::Some(MODULE::STORAGE),
            6 => ::std::option::Option::Some(MODULE::PD),
            7 => ::std::option::Option::Some(MODULE::METRIC),
            8 => ::std::option::Option::Some(MODULE::COPROCESSOR),
            9 => ::std::option::Option::Some(MODULE::SECURITY),
            10 => ::std::option::Option::Some(MODULE::IMPORT),
            _ => ::std::option::Option::None
        }
    }

    fn values() -> &'static [Self] {
        static values: &'static [MODULE] = &[
            MODULE::UNUSED,
            MODULE::KVDB,
            MODULE::RAFTDB,
            MODULE::READPOOL,
            MODULE::SERVER,
            MODULE::STORAGE,
            MODULE::PD,
            MODULE::METRIC,
            MODULE::COPROCESSOR,
            MODULE::SECURITY,
            MODULE::IMPORT,
        ];
        values
    }

    fn enum_descriptor_static(_: ::std::option::Option<MODULE>) -> &'static ::protobuf::reflect::EnumDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::EnumDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::EnumDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                ::protobuf::reflect::EnumDescriptor::new("MODULE", file_descriptor_proto())
            })
        }
    }
}

impl ::std::marker::Copy for MODULE {
}

impl ::std::default::Default for MODULE {
    fn default() -> Self {
        MODULE::UNUSED
    }
}

impl ::protobuf::reflect::ProtobufValue for MODULE {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Enum(self.descriptor())
    }
}

static file_descriptor_proto_data: &'static [u8] = b"\
    \n\rdebugpb.proto\x12\x07debugpb\x1a\reraftpb.proto\x1a\rkvrpcpb.proto\
    \x1a\x13raft_serverpb.proto\x1a\x14gogoproto/gogo.proto\"K\n\nGetRequest\
    \x12\x1b\n\x02db\x18\x01\x20\x01(\x0e2\x0b.debugpb.DBR\x02db\x12\x0e\n\
    \x02cf\x18\x02\x20\x01(\tR\x02cf\x12\x10\n\x03key\x18\x03\x20\x01(\x0cR\
    \x03key\"#\n\x0bGetResponse\x12\x14\n\x05value\x18\x01\x20\x01(\x0cR\x05\
    value\"J\n\x0eRaftLogRequest\x12\x1b\n\tregion_id\x18\x01\x20\x01(\x04R\
    \x08regionId\x12\x1b\n\tlog_index\x18\x02\x20\x01(\x04R\x08logIndex\"7\n\
    \x0fRaftLogResponse\x12$\n\x05entry\x18\x01\x20\x01(\x0b2\x0e.eraftpb.En\
    tryR\x05entry\"0\n\x11RegionInfoRequest\x12\x1b\n\tregion_id\x18\x01\x20\
    \x01(\x04R\x08regionId\"\xf5\x01\n\x12RegionInfoResponse\x12G\n\x10raft_\
    local_state\x18\x01\x20\x01(\x0b2\x1d.raft_serverpb.RaftLocalStateR\x0er\
    aftLocalState\x12G\n\x10raft_apply_state\x18\x02\x20\x01(\x0b2\x1d.raft_\
    serverpb.RaftApplyStateR\x0eraftApplyState\x12M\n\x12region_local_state\
    \x18\x03\x20\x01(\x0b2\x1f.raft_serverpb.RegionLocalStateR\x10regionLoca\
    lState\"B\n\x11RegionSizeRequest\x12\x1b\n\tregion_id\x18\x01\x20\x01(\
    \x04R\x08regionId\x12\x10\n\x03cfs\x18\x02\x20\x03(\tR\x03cfs\"~\n\x12Re\
    gionSizeResponse\x12;\n\x07entries\x18\x01\x20\x03(\x0b2!.debugpb.Region\
    SizeResponse.EntryR\x07entries\x1a+\n\x05Entry\x12\x0e\n\x02cf\x18\x01\
    \x20\x01(\tR\x02cf\x12\x12\n\x04size\x18\x02\x20\x01(\x04R\x04size\"Y\n\
    \x0fScanMvccRequest\x12\x19\n\x08from_key\x18\x01\x20\x01(\x0cR\x07fromK\
    ey\x12\x15\n\x06to_key\x18\x02\x20\x01(\x0cR\x05toKey\x12\x14\n\x05limit\
    \x18\x03\x20\x01(\x04R\x05limit\"K\n\x10ScanMvccResponse\x12\x10\n\x03ke\
    y\x18\x01\x20\x01(\x0cR\x03key\x12%\n\x04info\x18\x02\x20\x01(\x0b2\x11.\
    kvrpcpb.MvccInfoR\x04info\"\x89\x01\n\x0eCompactRequest\x12\x1b\n\x02db\
    \x18\x01\x20\x01(\x0e2\x0b.debugpb.DBR\x02db\x12\x0e\n\x02cf\x18\x02\x20\
    \x01(\tR\x02cf\x12\x19\n\x08from_key\x18\x03\x20\x01(\x0cR\x07fromKey\
    \x12\x15\n\x06to_key\x18\x04\x20\x01(\x0cR\x05toKey\x12\x18\n\x07threads\
    \x18\x05\x20\x01(\rR\x07threads\"\x11\n\x0fCompactResponse\"F\n\x16Injec\
    tFailPointRequest\x12\x12\n\x04name\x18\x01\x20\x01(\tR\x04name\x12\x18\
    \n\x07actions\x18\x02\x20\x01(\tR\x07actions\"\x19\n\x17InjectFailPointR\
    esponse\"-\n\x17RecoverFailPointRequest\x12\x12\n\x04name\x18\x01\x20\
    \x01(\tR\x04name\"\x1a\n\x18RecoverFailPointResponse\"\x17\n\x15ListFail\
    PointsRequest\"\x90\x01\n\x16ListFailPointsResponse\x12?\n\x07entries\
    \x18\x01\x20\x03(\x0b2%.debugpb.ListFailPointsResponse.EntryR\x07entries\
    \x1a5\n\x05Entry\x12\x12\n\x04name\x18\x01\x20\x01(\tR\x04name\x12\x18\n\
    \x07actions\x18\x02\x20\x01(\tR\x07actions\"%\n\x11GetMetricsRequest\x12\
    \x10\n\x03all\x18\x01\x20\x01(\x08R\x03all\"\x92\x01\n\x12GetMetricsResp\
    onse\x12\x1e\n\nprometheus\x18\x01\x20\x01(\tR\nprometheus\x12\x1d\n\nro\
    cksdb_kv\x18\x02\x20\x01(\tR\trocksdbKv\x12!\n\x0crocksdb_raft\x18\x03\
    \x20\x01(\tR\x0brocksdbRaft\x12\x1a\n\x08jemalloc\x18\x04\x20\x01(\tR\
    \x08jemalloc\"<\n\x1dRegionConsistencyCheckRequest\x12\x1b\n\tregion_id\
    \x18\x01\x20\x01(\x04R\x08regionId\"\x20\n\x1eRegionConsistencyCheckResp\
    onse\"\x86\x01\n\x17ModifyTikvConfigRequest\x12'\n\x06module\x18\x01\x20\
    \x01(\x0e2\x0f.debugpb.MODULER\x06module\x12\x1f\n\x0bconfig_name\x18\
    \x02\x20\x01(\tR\nconfigName\x12!\n\x0cconfig_value\x18\x03\x20\x01(\tR\
    \x0bconfigValue\"\x1a\n\x18ModifyTikvConfigResponse\"4\n\x08Property\x12\
    \x12\n\x04name\x18\x01\x20\x01(\tR\x04name\x12\x14\n\x05value\x18\x02\
    \x20\x01(\tR\x05value\"9\n\x1aGetRegionPropertiesRequest\x12\x1b\n\tregi\
    on_id\x18\x01\x20\x01(\x04R\x08regionId\"F\n\x1bGetRegionPropertiesRespo\
    nse\x12'\n\x05props\x18\x01\x20\x03(\x0b2\x11.debugpb.PropertyR\x05props\
    *#\n\x02DB\x12\x0b\n\x07INVALID\x10\0\x12\x06\n\x02KV\x10\x01\x12\x08\n\
    \x04RAFT\x10\x02*\x90\x01\n\x06MODULE\x12\n\n\x06UNUSED\x10\0\x12\x08\n\
    \x04KVDB\x10\x01\x12\n\n\x06RAFTDB\x10\x02\x12\x0c\n\x08READPOOL\x10\x03\
    \x12\n\n\x06SERVER\x10\x04\x12\x0b\n\x07STORAGE\x10\x05\x12\x06\n\x02PD\
    \x10\x06\x12\n\n\x06METRIC\x10\x07\x12\x0f\n\x0bCOPROCESSOR\x10\x08\x12\
    \x0c\n\x08SECURITY\x10\t\x12\n\n\x06IMPORT\x10\n2\x8f\x08\n\x05Debug\x12\
    2\n\x03Get\x12\x13.debugpb.GetRequest\x1a\x14.debugpb.GetResponse\"\0\
    \x12>\n\x07RaftLog\x12\x17.debugpb.RaftLogRequest\x1a\x18.debugpb.RaftLo\
    gResponse\"\0\x12G\n\nRegionInfo\x12\x1a.debugpb.RegionInfoRequest\x1a\
    \x1b.debugpb.RegionInfoResponse\"\0\x12G\n\nRegionSize\x12\x1a.debugpb.R\
    egionSizeRequest\x1a\x1b.debugpb.RegionSizeResponse\"\0\x12C\n\x08ScanMv\
    cc\x12\x18.debugpb.ScanMvccRequest\x1a\x19.debugpb.ScanMvccResponse\"\00\
    \x01\x12>\n\x07Compact\x12\x17.debugpb.CompactRequest\x1a\x18.debugpb.Co\
    mpactResponse\"\0\x12V\n\x0fInjectFailPoint\x12\x1f.debugpb.InjectFailPo\
    intRequest\x1a\x20.debugpb.InjectFailPointResponse\"\0\x12Y\n\x10Recover\
    FailPoint\x12\x20.debugpb.RecoverFailPointRequest\x1a!.debugpb.RecoverFa\
    ilPointResponse\"\0\x12S\n\x0eListFailPoints\x12\x1e.debugpb.ListFailPoi\
    ntsRequest\x1a\x1f.debugpb.ListFailPointsResponse\"\0\x12G\n\nGetMetrics\
    \x12\x1a.debugpb.GetMetricsRequest\x1a\x1b.debugpb.GetMetricsResponse\"\
    \0\x12k\n\x16CheckRegionConsistency\x12&.debugpb.RegionConsistencyCheckR\
    equest\x1a'.debugpb.RegionConsistencyCheckResponse\"\0\x12Y\n\x10ModifyT\
    ikvConfig\x12\x20.debugpb.ModifyTikvConfigRequest\x1a!.debugpb.ModifyTik\
    vConfigResponse\"\0\x12b\n\x13GetRegionProperties\x12#.debugpb.GetRegion\
    PropertiesRequest\x1a$.debugpb.GetRegionPropertiesResponse\"\0B&\n\x18co\
    m.pingcap.tikv.kvproto\xc8\xe2\x1e\x01\xe0\xe2\x1e\x01\xd0\xe2\x1e\x01J\
    \xa56\n\x07\x12\x05\0\0\xd8\x01\x01\n\x08\n\x01\x0c\x12\x03\0\0\x12\n\
    \x08\n\x01\x02\x12\x03\x01\x08\x0f\n\t\n\x02\x03\0\x12\x03\x03\x07\x16\n\
    \t\n\x02\x03\x01\x12\x03\x04\x07\x16\n\t\n\x02\x03\x02\x12\x03\x05\x07\
    \x1c\n\t\n\x02\x03\x03\x12\x03\x06\x07\x1d\n\x08\n\x01\x08\x12\x03\x08\0\
    $\n\x0b\n\x04\x08\xe7\x07\0\x12\x03\x08\0$\n\x0c\n\x05\x08\xe7\x07\0\x02\
    \x12\x03\x08\x07\x1c\n\r\n\x06\x08\xe7\x07\0\x02\0\x12\x03\x08\x07\x1c\n\
    \x0e\n\x07\x08\xe7\x07\0\x02\0\x01\x12\x03\x08\x08\x1b\n\x0c\n\x05\x08\
    \xe7\x07\0\x03\x12\x03\x08\x1f#\n\x08\n\x01\x08\x12\x03\t\0(\n\x0b\n\x04\
    \x08\xe7\x07\x01\x12\x03\t\0(\n\x0c\n\x05\x08\xe7\x07\x01\x02\x12\x03\t\
    \x07\x20\n\r\n\x06\x08\xe7\x07\x01\x02\0\x12\x03\t\x07\x20\n\x0e\n\x07\
    \x08\xe7\x07\x01\x02\0\x01\x12\x03\t\x08\x1f\n\x0c\n\x05\x08\xe7\x07\x01\
    \x03\x12\x03\t#'\n\x08\n\x01\x08\x12\x03\n\0*\n\x0b\n\x04\x08\xe7\x07\
    \x02\x12\x03\n\0*\n\x0c\n\x05\x08\xe7\x07\x02\x02\x12\x03\n\x07\"\n\r\n\
    \x06\x08\xe7\x07\x02\x02\0\x12\x03\n\x07\"\n\x0e\n\x07\x08\xe7\x07\x02\
    \x02\0\x01\x12\x03\n\x08!\n\x0c\n\x05\x08\xe7\x07\x02\x03\x12\x03\n%)\n\
    \x08\n\x01\x08\x12\x03\x0c\01\n\x0b\n\x04\x08\xe7\x07\x03\x12\x03\x0c\01\
    \n\x0c\n\x05\x08\xe7\x07\x03\x02\x12\x03\x0c\x07\x13\n\r\n\x06\x08\xe7\
    \x07\x03\x02\0\x12\x03\x0c\x07\x13\n\x0e\n\x07\x08\xe7\x07\x03\x02\0\x01\
    \x12\x03\x0c\x07\x13\n\x0c\n\x05\x08\xe7\x07\x03\x07\x12\x03\x0c\x160\n\
    \xe3\x02\n\x02\x06\0\x12\x04\x17\0@\x01\x1a\xd6\x02\x20Debug\x20service\
    \x20for\x20TiKV.\n\n\x20Errors\x20are\x20defined\x20as\x20follow:\n\x20\
    \x20\x20-\x20OK:\x20Okay,\x20we\x20are\x20good!\n\x20\x20\x20-\x20UNKNOW\
    N:\x20For\x20unknown\x20error.\n\x20\x20\x20-\x20INVALID_ARGUMENT:\x20So\
    mething\x20goes\x20wrong\x20within\x20requests.\n\x20\x20\x20-\x20NOT_FO\
    UND:\x20It\x20is\x20key\x20or\x20region\x20not\x20found,\x20it's\x20base\
    d\x20on\x20context,\x20detailed\n\x20\x20\x20\x20\x20\x20\x20\x20\x20\
    \x20\x20\x20\x20\x20\x20\x20reason\x20can\x20be\x20found\x20in\x20grpc\
    \x20message.\n\x20Note:\x20It\x20bypasses\x20raft\x20layer.\n\n\n\n\x03\
    \x06\0\x01\x12\x03\x17\x08\r\nd\n\x04\x06\0\x02\0\x12\x03\x1a\x040\x1aW\
    \x20Read\x20a\x20value\x20arbitrarily\x20for\x20a\x20key.\n\x20Note:\x20\
    Server\x20uses\x20key\x20directly\x20w/o\x20any\x20encoding.\n\n\x0c\n\
    \x05\x06\0\x02\0\x01\x12\x03\x1a\x08\x0b\n\x0c\n\x05\x06\0\x02\0\x02\x12\
    \x03\x1a\x0c\x16\n\x0c\n\x05\x06\0\x02\0\x03\x12\x03\x1a!,\n\x1e\n\x04\
    \x06\0\x02\x01\x12\x03\x1d\x04<\x1a\x11\x20Read\x20raft\x20info.\n\n\x0c\
    \n\x05\x06\0\x02\x01\x01\x12\x03\x1d\x08\x0f\n\x0c\n\x05\x06\0\x02\x01\
    \x02\x12\x03\x1d\x10\x1e\n\x0c\n\x05\x06\0\x02\x01\x03\x12\x03\x1d)8\n\
    \x0b\n\x04\x06\0\x02\x02\x12\x03\x1e\x04E\n\x0c\n\x05\x06\0\x02\x02\x01\
    \x12\x03\x1e\x08\x12\n\x0c\n\x05\x06\0\x02\x02\x02\x12\x03\x1e\x13$\n\
    \x0c\n\x05\x06\0\x02\x02\x03\x12\x03\x1e/A\nf\n\x04\x06\0\x02\x03\x12\
    \x03\"\x04E\x1aY\x20Calculate\x20size\x20of\x20a\x20region.\n\x20Note:\
    \x20DO\x20NOT\x20CALL\x20IT\x20IN\x20PRODUCTION,\x20it's\x20really\x20ex\
    pensive.\n\n\x0c\n\x05\x06\0\x02\x03\x01\x12\x03\"\x08\x12\n\x0c\n\x05\
    \x06\0\x02\x03\x02\x12\x03\"\x13$\n\x0c\n\x05\x06\0\x02\x03\x03\x12\x03\
    \"/A\n\x95\x01\n\x04\x06\0\x02\x04\x12\x03'\x04F\x1a\x87\x01\x20Scan\x20\
    a\x20specific\x20range.\n\x20Note:\x20DO\x20NOT\x20CALL\x20IT\x20IN\x20P\
    RODUCTION,\x20it's\x20really\x20expensive.\n\x20\x20\x20\x20\x20\x20\x20\
    Server\x20uses\x20keys\x20directly\x20w/o\x20any\x20encoding.\n\n\x0c\n\
    \x05\x06\0\x02\x04\x01\x12\x03'\x08\x10\n\x0c\n\x05\x06\0\x02\x04\x02\
    \x12\x03'\x11\x20\n\x0c\n\x05\x06\0\x02\x04\x06\x12\x03'+1\n\x0c\n\x05\
    \x06\0\x02\x04\x03\x12\x03'2B\no\n\x04\x06\0\x02\x05\x12\x03+\x04<\x1ab\
    \x20Compact\x20a\x20column\x20family\x20in\x20a\x20specified\x20range.\n\
    \x20Note:\x20Server\x20uses\x20keys\x20directly\x20w/o\x20any\x20encodin\
    g.\n\n\x0c\n\x05\x06\0\x02\x05\x01\x12\x03+\x08\x0f\n\x0c\n\x05\x06\0\
    \x02\x05\x02\x12\x03+\x10\x1e\n\x0c\n\x05\x06\0\x02\x05\x03\x12\x03+)8\n\
    l\n\x04\x06\0\x02\x06\x12\x03/\x04T\x1a_\x20Inject\x20a\x20fail\x20point\
    .\x20Currently,\x20it's\x20only\x20used\x20in\x20tests.\n\x20Note:\x20DO\
    \x20NOT\x20CALL\x20IT\x20IN\x20PRODUCTION.\n\n\x0c\n\x05\x06\0\x02\x06\
    \x01\x12\x03/\x08\x17\n\x0c\n\x05\x06\0\x02\x06\x02\x12\x03/\x18.\n\x0c\
    \n\x05\x06\0\x02\x06\x03\x12\x03/9P\n)\n\x04\x06\0\x02\x07\x12\x031\x04W\
    \x1a\x1c\x20Recover\x20from\x20a\x20fail\x20point.\n\n\x0c\n\x05\x06\0\
    \x02\x07\x01\x12\x031\x08\x18\n\x0c\n\x05\x06\0\x02\x07\x02\x12\x031\x19\
    0\n\x0c\n\x05\x06\0\x02\x07\x03\x12\x031;S\n$\n\x04\x06\0\x02\x08\x12\
    \x033\x04Q\x1a\x17\x20List\x20all\x20fail\x20points.\n\n\x0c\n\x05\x06\0\
    \x02\x08\x01\x12\x033\x08\x16\n\x0c\n\x05\x06\0\x02\x08\x02\x12\x033\x17\
    ,\n\x0c\n\x05\x06\0\x02\x08\x03\x12\x0337M\n\x1a\n\x04\x06\0\x02\t\x12\
    \x036\x04D\x1a\r\x20Get\x20Metrics\n\n\x0c\n\x05\x06\0\x02\t\x01\x12\x03\
    6\x08\x12\n\x0c\n\x05\x06\0\x02\t\x02\x12\x036\x13$\n\x0c\n\x05\x06\0\
    \x02\t\x03\x12\x036/A\n2\n\x04\x06\0\x02\n\x12\x039\x04i\x1a%\x20Do\x20a\
    \x20consistent\x20check\x20for\x20a\x20region.\n\n\x0c\n\x05\x06\0\x02\n\
    \x01\x12\x039\x08\x1e\n\x0c\n\x05\x06\0\x02\n\x02\x12\x039\x1f<\n\x0c\n\
    \x05\x06\0\x02\n\x03\x12\x039Ge\n/\n\x04\x06\0\x02\x0b\x12\x03<\x04W\x1a\
    \"\x20dynamically\x20modify\x20tikv's\x20config\n\n\x0c\n\x05\x06\0\x02\
    \x0b\x01\x12\x03<\x08\x18\n\x0c\n\x05\x06\0\x02\x0b\x02\x12\x03<\x190\n\
    \x0c\n\x05\x06\0\x02\x0b\x03\x12\x03<;S\n$\n\x04\x06\0\x02\x0c\x12\x03?\
    \x04`\x1a\x17\x20Get\x20region\x20properties\n\n\x0c\n\x05\x06\0\x02\x0c\
    \x01\x12\x03?\x08\x1b\n\x0c\n\x05\x06\0\x02\x0c\x02\x12\x03?\x1c6\n\x0c\
    \n\x05\x06\0\x02\x0c\x03\x12\x03?A\\\n\n\n\x02\x05\0\x12\x04B\0F\x01\n\n\
    \n\x03\x05\0\x01\x12\x03B\x05\x07\n\x0b\n\x04\x05\0\x02\0\x12\x03C\x04\
    \x10\n\x0c\n\x05\x05\0\x02\0\x01\x12\x03C\x04\x0b\n\x0c\n\x05\x05\0\x02\
    \0\x02\x12\x03C\x0e\x0f\n\x0b\n\x04\x05\0\x02\x01\x12\x03D\x04\x0b\n\x0c\
    \n\x05\x05\0\x02\x01\x01\x12\x03D\x04\x06\n\x0c\n\x05\x05\0\x02\x01\x02\
    \x12\x03D\t\n\n\x0b\n\x04\x05\0\x02\x02\x12\x03E\x04\r\n\x0c\n\x05\x05\0\
    \x02\x02\x01\x12\x03E\x04\x08\n\x0c\n\x05\x05\0\x02\x02\x02\x12\x03E\x0b\
    \x0c\n\n\n\x02\x05\x01\x12\x04H\0T\x01\n\n\n\x03\x05\x01\x01\x12\x03H\
    \x05\x0b\n\x0b\n\x04\x05\x01\x02\0\x12\x03I\x04\x0f\n\x0c\n\x05\x05\x01\
    \x02\0\x01\x12\x03I\x04\n\n\x0c\n\x05\x05\x01\x02\0\x02\x12\x03I\r\x0e\n\
    \x0b\n\x04\x05\x01\x02\x01\x12\x03J\x04\r\n\x0c\n\x05\x05\x01\x02\x01\
    \x01\x12\x03J\x04\x08\n\x0c\n\x05\x05\x01\x02\x01\x02\x12\x03J\x0b\x0c\n\
    \x0b\n\x04\x05\x01\x02\x02\x12\x03K\x04\x0f\n\x0c\n\x05\x05\x01\x02\x02\
    \x01\x12\x03K\x04\n\n\x0c\n\x05\x05\x01\x02\x02\x02\x12\x03K\r\x0e\n\x0b\
    \n\x04\x05\x01\x02\x03\x12\x03L\x04\x11\n\x0c\n\x05\x05\x01\x02\x03\x01\
    \x12\x03L\x04\x0c\n\x0c\n\x05\x05\x01\x02\x03\x02\x12\x03L\x0f\x10\n\x0b\
    \n\x04\x05\x01\x02\x04\x12\x03M\x04\x0f\n\x0c\n\x05\x05\x01\x02\x04\x01\
    \x12\x03M\x04\n\n\x0c\n\x05\x05\x01\x02\x04\x02\x12\x03M\r\x0e\n\x0b\n\
    \x04\x05\x01\x02\x05\x12\x03N\x04\x10\n\x0c\n\x05\x05\x01\x02\x05\x01\
    \x12\x03N\x04\x0b\n\x0c\n\x05\x05\x01\x02\x05\x02\x12\x03N\x0e\x0f\n\x0b\
    \n\x04\x05\x01\x02\x06\x12\x03O\x04\x0b\n\x0c\n\x05\x05\x01\x02\x06\x01\
    \x12\x03O\x04\x06\n\x0c\n\x05\x05\x01\x02\x06\x02\x12\x03O\t\n\n\x0b\n\
    \x04\x05\x01\x02\x07\x12\x03P\x04\x0f\n\x0c\n\x05\x05\x01\x02\x07\x01\
    \x12\x03P\x04\n\n\x0c\n\x05\x05\x01\x02\x07\x02\x12\x03P\r\x0e\n\x0b\n\
    \x04\x05\x01\x02\x08\x12\x03Q\x04\x14\n\x0c\n\x05\x05\x01\x02\x08\x01\
    \x12\x03Q\x04\x0f\n\x0c\n\x05\x05\x01\x02\x08\x02\x12\x03Q\x12\x13\n\x0b\
    \n\x04\x05\x01\x02\t\x12\x03R\x04\x11\n\x0c\n\x05\x05\x01\x02\t\x01\x12\
    \x03R\x04\x0c\n\x0c\n\x05\x05\x01\x02\t\x02\x12\x03R\x0f\x10\n\x0b\n\x04\
    \x05\x01\x02\n\x12\x03S\x04\x10\n\x0c\n\x05\x05\x01\x02\n\x01\x12\x03S\
    \x04\n\n\x0c\n\x05\x05\x01\x02\n\x02\x12\x03S\r\x0f\n\n\n\x02\x04\0\x12\
    \x04V\0Z\x01\n\n\n\x03\x04\0\x01\x12\x03V\x08\x12\n\x0b\n\x04\x04\0\x02\
    \0\x12\x03W\x04\x0e\n\r\n\x05\x04\0\x02\0\x04\x12\x04W\x04V\x14\n\x0c\n\
    \x05\x04\0\x02\0\x06\x12\x03W\x04\x06\n\x0c\n\x05\x04\0\x02\0\x01\x12\
    \x03W\x07\t\n\x0c\n\x05\x04\0\x02\0\x03\x12\x03W\x0c\r\n\x0b\n\x04\x04\0\
    \x02\x01\x12\x03X\x04\x12\n\r\n\x05\x04\0\x02\x01\x04\x12\x04X\x04W\x0e\
    \n\x0c\n\x05\x04\0\x02\x01\x05\x12\x03X\x04\n\n\x0c\n\x05\x04\0\x02\x01\
    \x01\x12\x03X\x0b\r\n\x0c\n\x05\x04\0\x02\x01\x03\x12\x03X\x10\x11\n\x0b\
    \n\x04\x04\0\x02\x02\x12\x03Y\x04\x12\n\r\n\x05\x04\0\x02\x02\x04\x12\
    \x04Y\x04X\x12\n\x0c\n\x05\x04\0\x02\x02\x05\x12\x03Y\x04\t\n\x0c\n\x05\
    \x04\0\x02\x02\x01\x12\x03Y\n\r\n\x0c\n\x05\x04\0\x02\x02\x03\x12\x03Y\
    \x10\x11\n\n\n\x02\x04\x01\x12\x04\\\0^\x01\n\n\n\x03\x04\x01\x01\x12\
    \x03\\\x08\x13\n\x0b\n\x04\x04\x01\x02\0\x12\x03]\x04\x14\n\r\n\x05\x04\
    \x01\x02\0\x04\x12\x04]\x04\\\x15\n\x0c\n\x05\x04\x01\x02\0\x05\x12\x03]\
    \x04\t\n\x0c\n\x05\x04\x01\x02\0\x01\x12\x03]\n\x0f\n\x0c\n\x05\x04\x01\
    \x02\0\x03\x12\x03]\x12\x13\n\n\n\x02\x04\x02\x12\x04`\0c\x01\n\n\n\x03\
    \x04\x02\x01\x12\x03`\x08\x16\n\x0b\n\x04\x04\x02\x02\0\x12\x03a\x04\x19\
    \n\r\n\x05\x04\x02\x02\0\x04\x12\x04a\x04`\x18\n\x0c\n\x05\x04\x02\x02\0\
    \x05\x12\x03a\x04\n\n\x0c\n\x05\x04\x02\x02\0\x01\x12\x03a\x0b\x14\n\x0c\
    \n\x05\x04\x02\x02\0\x03\x12\x03a\x17\x18\n\x0b\n\x04\x04\x02\x02\x01\
    \x12\x03b\x04\x19\n\r\n\x05\x04\x02\x02\x01\x04\x12\x04b\x04a\x19\n\x0c\
    \n\x05\x04\x02\x02\x01\x05\x12\x03b\x04\n\n\x0c\n\x05\x04\x02\x02\x01\
    \x01\x12\x03b\x0b\x14\n\x0c\n\x05\x04\x02\x02\x01\x03\x12\x03b\x17\x18\n\
    \n\n\x02\x04\x03\x12\x04e\0g\x01\n\n\n\x03\x04\x03\x01\x12\x03e\x08\x17\
    \n\x0b\n\x04\x04\x03\x02\0\x12\x03f\x04\x1c\n\r\n\x05\x04\x03\x02\0\x04\
    \x12\x04f\x04e\x19\n\x0c\n\x05\x04\x03\x02\0\x06\x12\x03f\x04\x11\n\x0c\
    \n\x05\x04\x03\x02\0\x01\x12\x03f\x12\x17\n\x0c\n\x05\x04\x03\x02\0\x03\
    \x12\x03f\x1a\x1b\n\n\n\x02\x04\x04\x12\x04i\0k\x01\n\n\n\x03\x04\x04\
    \x01\x12\x03i\x08\x19\n\x0b\n\x04\x04\x04\x02\0\x12\x03j\x04\x19\n\r\n\
    \x05\x04\x04\x02\0\x04\x12\x04j\x04i\x1b\n\x0c\n\x05\x04\x04\x02\0\x05\
    \x12\x03j\x04\n\n\x0c\n\x05\x04\x04\x02\0\x01\x12\x03j\x0b\x14\n\x0c\n\
    \x05\x04\x04\x02\0\x03\x12\x03j\x17\x18\n\n\n\x02\x04\x05\x12\x04m\0q\
    \x01\n\n\n\x03\x04\x05\x01\x12\x03m\x08\x1a\n\x0b\n\x04\x04\x05\x02\0\
    \x12\x03n\x046\n\r\n\x05\x04\x05\x02\0\x04\x12\x04n\x04m\x1c\n\x0c\n\x05\
    \x04\x05\x02\0\x06\x12\x03n\x04\x20\n\x0c\n\x05\x04\x05\x02\0\x01\x12\
    \x03n!1\n\x0c\n\x05\x04\x05\x02\0\x03\x12\x03n45\n\x0b\n\x04\x04\x05\x02\
    \x01\x12\x03o\x046\n\r\n\x05\x04\x05\x02\x01\x04\x12\x04o\x04n6\n\x0c\n\
    \x05\x04\x05\x02\x01\x06\x12\x03o\x04\x20\n\x0c\n\x05\x04\x05\x02\x01\
    \x01\x12\x03o!1\n\x0c\n\x05\x04\x05\x02\x01\x03\x12\x03o45\n\x0b\n\x04\
    \x04\x05\x02\x02\x12\x03p\x04:\n\r\n\x05\x04\x05\x02\x02\x04\x12\x04p\
    \x04o6\n\x0c\n\x05\x04\x05\x02\x02\x06\x12\x03p\x04\"\n\x0c\n\x05\x04\
    \x05\x02\x02\x01\x12\x03p#5\n\x0c\n\x05\x04\x05\x02\x02\x03\x12\x03p89\n\
    \n\n\x02\x04\x06\x12\x04s\0v\x01\n\n\n\x03\x04\x06\x01\x12\x03s\x08\x19\
    \n\x0b\n\x04\x04\x06\x02\0\x12\x03t\x04\x19\n\r\n\x05\x04\x06\x02\0\x04\
    \x12\x04t\x04s\x1b\n\x0c\n\x05\x04\x06\x02\0\x05\x12\x03t\x04\n\n\x0c\n\
    \x05\x04\x06\x02\0\x01\x12\x03t\x0b\x14\n\x0c\n\x05\x04\x06\x02\0\x03\
    \x12\x03t\x17\x18\n\x0b\n\x04\x04\x06\x02\x01\x12\x03u\x04\x1c\n\x0c\n\
    \x05\x04\x06\x02\x01\x04\x12\x03u\x04\x0c\n\x0c\n\x05\x04\x06\x02\x01\
    \x05\x12\x03u\r\x13\n\x0c\n\x05\x04\x06\x02\x01\x01\x12\x03u\x14\x17\n\
    \x0c\n\x05\x04\x06\x02\x01\x03\x12\x03u\x1a\x1b\n\n\n\x02\x04\x07\x12\
    \x04x\0\x7f\x01\n\n\n\x03\x04\x07\x01\x12\x03x\x08\x1a\n\x0c\n\x04\x04\
    \x07\x03\0\x12\x04y\x04|\x05\n\x0c\n\x05\x04\x07\x03\0\x01\x12\x03y\x0c\
    \x11\n\r\n\x06\x04\x07\x03\0\x02\0\x12\x03z\x08\x16\n\x0f\n\x07\x04\x07\
    \x03\0\x02\0\x04\x12\x04z\x08y\x13\n\x0e\n\x07\x04\x07\x03\0\x02\0\x05\
    \x12\x03z\x08\x0e\n\x0e\n\x07\x04\x07\x03\0\x02\0\x01\x12\x03z\x0f\x11\n\
    \x0e\n\x07\x04\x07\x03\0\x02\0\x03\x12\x03z\x14\x15\n\r\n\x06\x04\x07\
    \x03\0\x02\x01\x12\x03{\x08\x18\n\x0f\n\x07\x04\x07\x03\0\x02\x01\x04\
    \x12\x04{\x08z\x16\n\x0e\n\x07\x04\x07\x03\0\x02\x01\x05\x12\x03{\x08\
    \x0e\n\x0e\n\x07\x04\x07\x03\0\x02\x01\x01\x12\x03{\x0f\x13\n\x0e\n\x07\
    \x04\x07\x03\0\x02\x01\x03\x12\x03{\x16\x17\n\x0b\n\x04\x04\x07\x02\0\
    \x12\x03~\x04\x1f\n\x0c\n\x05\x04\x07\x02\0\x04\x12\x03~\x04\x0c\n\x0c\n\
    \x05\x04\x07\x02\0\x06\x12\x03~\r\x12\n\x0c\n\x05\x04\x07\x02\0\x01\x12\
    \x03~\x13\x1a\n\x0c\n\x05\x04\x07\x02\0\x03\x12\x03~\x1d\x1e\n\x0c\n\x02\
    \x04\x08\x12\x06\x81\x01\0\x85\x01\x01\n\x0b\n\x03\x04\x08\x01\x12\x04\
    \x81\x01\x08\x17\n\x0c\n\x04\x04\x08\x02\0\x12\x04\x82\x01\x04\x17\n\x0f\
    \n\x05\x04\x08\x02\0\x04\x12\x06\x82\x01\x04\x81\x01\x19\n\r\n\x05\x04\
    \x08\x02\0\x05\x12\x04\x82\x01\x04\t\n\r\n\x05\x04\x08\x02\0\x01\x12\x04\
    \x82\x01\n\x12\n\r\n\x05\x04\x08\x02\0\x03\x12\x04\x82\x01\x15\x16\n\x0c\
    \n\x04\x04\x08\x02\x01\x12\x04\x83\x01\x04\x15\n\x0f\n\x05\x04\x08\x02\
    \x01\x04\x12\x06\x83\x01\x04\x82\x01\x17\n\r\n\x05\x04\x08\x02\x01\x05\
    \x12\x04\x83\x01\x04\t\n\r\n\x05\x04\x08\x02\x01\x01\x12\x04\x83\x01\n\
    \x10\n\r\n\x05\x04\x08\x02\x01\x03\x12\x04\x83\x01\x13\x14\n\x0c\n\x04\
    \x04\x08\x02\x02\x12\x04\x84\x01\x04\x15\n\x0f\n\x05\x04\x08\x02\x02\x04\
    \x12\x06\x84\x01\x04\x83\x01\x15\n\r\n\x05\x04\x08\x02\x02\x05\x12\x04\
    \x84\x01\x04\n\n\r\n\x05\x04\x08\x02\x02\x01\x12\x04\x84\x01\x0b\x10\n\r\
    \n\x05\x04\x08\x02\x02\x03\x12\x04\x84\x01\x13\x14\n\x0c\n\x02\x04\t\x12\
    \x06\x87\x01\0\x8a\x01\x01\n\x0b\n\x03\x04\t\x01\x12\x04\x87\x01\x08\x18\
    \n\x0c\n\x04\x04\t\x02\0\x12\x04\x88\x01\x04\x12\n\x0f\n\x05\x04\t\x02\0\
    \x04\x12\x06\x88\x01\x04\x87\x01\x1a\n\r\n\x05\x04\t\x02\0\x05\x12\x04\
    \x88\x01\x04\t\n\r\n\x05\x04\t\x02\0\x01\x12\x04\x88\x01\n\r\n\r\n\x05\
    \x04\t\x02\0\x03\x12\x04\x88\x01\x10\x11\n\x0c\n\x04\x04\t\x02\x01\x12\
    \x04\x89\x01\x04\x1e\n\x0f\n\x05\x04\t\x02\x01\x04\x12\x06\x89\x01\x04\
    \x88\x01\x12\n\r\n\x05\x04\t\x02\x01\x06\x12\x04\x89\x01\x04\x14\n\r\n\
    \x05\x04\t\x02\x01\x01\x12\x04\x89\x01\x15\x19\n\r\n\x05\x04\t\x02\x01\
    \x03\x12\x04\x89\x01\x1c\x1d\n\x0c\n\x02\x04\n\x12\x06\x8c\x01\0\x92\x01\
    \x01\n\x0b\n\x03\x04\n\x01\x12\x04\x8c\x01\x08\x16\n\x0c\n\x04\x04\n\x02\
    \0\x12\x04\x8d\x01\x04\x0e\n\x0f\n\x05\x04\n\x02\0\x04\x12\x06\x8d\x01\
    \x04\x8c\x01\x18\n\r\n\x05\x04\n\x02\0\x06\x12\x04\x8d\x01\x04\x06\n\r\n\
    \x05\x04\n\x02\0\x01\x12\x04\x8d\x01\x07\t\n\r\n\x05\x04\n\x02\0\x03\x12\
    \x04\x8d\x01\x0c\r\n\x0c\n\x04\x04\n\x02\x01\x12\x04\x8e\x01\x04\x12\n\
    \x0f\n\x05\x04\n\x02\x01\x04\x12\x06\x8e\x01\x04\x8d\x01\x0e\n\r\n\x05\
    \x04\n\x02\x01\x05\x12\x04\x8e\x01\x04\n\n\r\n\x05\x04\n\x02\x01\x01\x12\
    \x04\x8e\x01\x0b\r\n\r\n\x05\x04\n\x02\x01\x03\x12\x04\x8e\x01\x10\x11\n\
    \x0c\n\x04\x04\n\x02\x02\x12\x04\x8f\x01\x04\x17\n\x0f\n\x05\x04\n\x02\
    \x02\x04\x12\x06\x8f\x01\x04\x8e\x01\x12\n\r\n\x05\x04\n\x02\x02\x05\x12\
    \x04\x8f\x01\x04\t\n\r\n\x05\x04\n\x02\x02\x01\x12\x04\x8f\x01\n\x12\n\r\
    \n\x05\x04\n\x02\x02\x03\x12\x04\x8f\x01\x15\x16\n\x0c\n\x04\x04\n\x02\
    \x03\x12\x04\x90\x01\x04\x15\n\x0f\n\x05\x04\n\x02\x03\x04\x12\x06\x90\
    \x01\x04\x8f\x01\x17\n\r\n\x05\x04\n\x02\x03\x05\x12\x04\x90\x01\x04\t\n\
    \r\n\x05\x04\n\x02\x03\x01\x12\x04\x90\x01\n\x10\n\r\n\x05\x04\n\x02\x03\
    \x03\x12\x04\x90\x01\x13\x14\n\x0c\n\x04\x04\n\x02\x04\x12\x04\x91\x01\
    \x04\x17\n\x0f\n\x05\x04\n\x02\x04\x04\x12\x06\x91\x01\x04\x90\x01\x15\n\
    \r\n\x05\x04\n\x02\x04\x05\x12\x04\x91\x01\x04\n\n\r\n\x05\x04\n\x02\x04\
    \x01\x12\x04\x91\x01\x0b\x12\n\r\n\x05\x04\n\x02\x04\x03\x12\x04\x91\x01\
    \x15\x16\n\x0c\n\x02\x04\x0b\x12\x06\x94\x01\0\x95\x01\x01\n\x0b\n\x03\
    \x04\x0b\x01\x12\x04\x94\x01\x08\x17\n\x0c\n\x02\x04\x0c\x12\x06\x97\x01\
    \0\x9a\x01\x01\n\x0b\n\x03\x04\x0c\x01\x12\x04\x97\x01\x08\x1e\n\x0c\n\
    \x04\x04\x0c\x02\0\x12\x04\x98\x01\x04\x14\n\x0f\n\x05\x04\x0c\x02\0\x04\
    \x12\x06\x98\x01\x04\x97\x01\x20\n\r\n\x05\x04\x0c\x02\0\x05\x12\x04\x98\
    \x01\x04\n\n\r\n\x05\x04\x0c\x02\0\x01\x12\x04\x98\x01\x0b\x0f\n\r\n\x05\
    \x04\x0c\x02\0\x03\x12\x04\x98\x01\x12\x13\n\x0c\n\x04\x04\x0c\x02\x01\
    \x12\x04\x99\x01\x04\x17\n\x0f\n\x05\x04\x0c\x02\x01\x04\x12\x06\x99\x01\
    \x04\x98\x01\x14\n\r\n\x05\x04\x0c\x02\x01\x05\x12\x04\x99\x01\x04\n\n\r\
    \n\x05\x04\x0c\x02\x01\x01\x12\x04\x99\x01\x0b\x12\n\r\n\x05\x04\x0c\x02\
    \x01\x03\x12\x04\x99\x01\x15\x16\n\x0c\n\x02\x04\r\x12\x06\x9c\x01\0\x9d\
    \x01\x01\n\x0b\n\x03\x04\r\x01\x12\x04\x9c\x01\x08\x1f\n\x0c\n\x02\x04\
    \x0e\x12\x06\x9f\x01\0\xa1\x01\x01\n\x0b\n\x03\x04\x0e\x01\x12\x04\x9f\
    \x01\x08\x1f\n\x0c\n\x04\x04\x0e\x02\0\x12\x04\xa0\x01\x04\x14\n\x0f\n\
    \x05\x04\x0e\x02\0\x04\x12\x06\xa0\x01\x04\x9f\x01!\n\r\n\x05\x04\x0e\
    \x02\0\x05\x12\x04\xa0\x01\x04\n\n\r\n\x05\x04\x0e\x02\0\x01\x12\x04\xa0\
    \x01\x0b\x0f\n\r\n\x05\x04\x0e\x02\0\x03\x12\x04\xa0\x01\x12\x13\n\x0c\n\
    \x02\x04\x0f\x12\x06\xa3\x01\0\xa4\x01\x01\n\x0b\n\x03\x04\x0f\x01\x12\
    \x04\xa3\x01\x08\x20\n\x0c\n\x02\x04\x10\x12\x06\xa6\x01\0\xa7\x01\x01\n\
    \x0b\n\x03\x04\x10\x01\x12\x04\xa6\x01\x08\x1d\n\x0c\n\x02\x04\x11\x12\
    \x06\xa9\x01\0\xb0\x01\x01\n\x0b\n\x03\x04\x11\x01\x12\x04\xa9\x01\x08\
    \x1e\n\x0e\n\x04\x04\x11\x03\0\x12\x06\xaa\x01\x04\xad\x01\x05\n\r\n\x05\
    \x04\x11\x03\0\x01\x12\x04\xaa\x01\x0c\x11\n\x0e\n\x06\x04\x11\x03\0\x02\
    \0\x12\x04\xab\x01\x08\x18\n\x11\n\x07\x04\x11\x03\0\x02\0\x04\x12\x06\
    \xab\x01\x08\xaa\x01\x13\n\x0f\n\x07\x04\x11\x03\0\x02\0\x05\x12\x04\xab\
    \x01\x08\x0e\n\x0f\n\x07\x04\x11\x03\0\x02\0\x01\x12\x04\xab\x01\x0f\x13\
    \n\x0f\n\x07\x04\x11\x03\0\x02\0\x03\x12\x04\xab\x01\x16\x17\n\x0e\n\x06\
    \x04\x11\x03\0\x02\x01\x12\x04\xac\x01\x08\x1b\n\x11\n\x07\x04\x11\x03\0\
    \x02\x01\x04\x12\x06\xac\x01\x08\xab\x01\x18\n\x0f\n\x07\x04\x11\x03\0\
    \x02\x01\x05\x12\x04\xac\x01\x08\x0e\n\x0f\n\x07\x04\x11\x03\0\x02\x01\
    \x01\x12\x04\xac\x01\x0f\x16\n\x0f\n\x07\x04\x11\x03\0\x02\x01\x03\x12\
    \x04\xac\x01\x19\x1a\n\x0c\n\x04\x04\x11\x02\0\x12\x04\xaf\x01\x04\x1f\n\
    \r\n\x05\x04\x11\x02\0\x04\x12\x04\xaf\x01\x04\x0c\n\r\n\x05\x04\x11\x02\
    \0\x06\x12\x04\xaf\x01\r\x12\n\r\n\x05\x04\x11\x02\0\x01\x12\x04\xaf\x01\
    \x13\x1a\n\r\n\x05\x04\x11\x02\0\x03\x12\x04\xaf\x01\x1d\x1e\n\x0c\n\x02\
    \x04\x12\x12\x06\xb2\x01\0\xb4\x01\x01\n\x0b\n\x03\x04\x12\x01\x12\x04\
    \xb2\x01\x08\x19\n\x0c\n\x04\x04\x12\x02\0\x12\x04\xb3\x01\x03\x10\n\x0f\
    \n\x05\x04\x12\x02\0\x04\x12\x06\xb3\x01\x03\xb2\x01\x1b\n\r\n\x05\x04\
    \x12\x02\0\x05\x12\x04\xb3\x01\x03\x07\n\r\n\x05\x04\x12\x02\0\x01\x12\
    \x04\xb3\x01\x08\x0b\n\r\n\x05\x04\x12\x02\0\x03\x12\x04\xb3\x01\x0e\x0f\
    \n\x0c\n\x02\x04\x13\x12\x06\xb6\x01\0\xbb\x01\x01\n\x0b\n\x03\x04\x13\
    \x01\x12\x04\xb6\x01\x08\x1a\n\x0c\n\x04\x04\x13\x02\0\x12\x04\xb7\x01\
    \x03\x19\n\x0f\n\x05\x04\x13\x02\0\x04\x12\x06\xb7\x01\x03\xb6\x01\x1c\n\
    \r\n\x05\x04\x13\x02\0\x05\x12\x04\xb7\x01\x03\t\n\r\n\x05\x04\x13\x02\0\
    \x01\x12\x04\xb7\x01\n\x14\n\r\n\x05\x04\x13\x02\0\x03\x12\x04\xb7\x01\
    \x17\x18\n\x0c\n\x04\x04\x13\x02\x01\x12\x04\xb8\x01\x03\x19\n\x0f\n\x05\
    \x04\x13\x02\x01\x04\x12\x06\xb8\x01\x03\xb7\x01\x19\n\r\n\x05\x04\x13\
    \x02\x01\x05\x12\x04\xb8\x01\x03\t\n\r\n\x05\x04\x13\x02\x01\x01\x12\x04\
    \xb8\x01\n\x14\n\r\n\x05\x04\x13\x02\x01\x03\x12\x04\xb8\x01\x17\x18\n\
    \x0c\n\x04\x04\x13\x02\x02\x12\x04\xb9\x01\x03\x1b\n\x0f\n\x05\x04\x13\
    \x02\x02\x04\x12\x06\xb9\x01\x03\xb8\x01\x19\n\r\n\x05\x04\x13\x02\x02\
    \x05\x12\x04\xb9\x01\x03\t\n\r\n\x05\x04\x13\x02\x02\x01\x12\x04\xb9\x01\
    \n\x16\n\r\n\x05\x04\x13\x02\x02\x03\x12\x04\xb9\x01\x19\x1a\n\x0c\n\x04\
    \x04\x13\x02\x03\x12\x04\xba\x01\x03\x17\n\x0f\n\x05\x04\x13\x02\x03\x04\
    \x12\x06\xba\x01\x03\xb9\x01\x1b\n\r\n\x05\x04\x13\x02\x03\x05\x12\x04\
    \xba\x01\x03\t\n\r\n\x05\x04\x13\x02\x03\x01\x12\x04\xba\x01\n\x12\n\r\n\
    \x05\x04\x13\x02\x03\x03\x12\x04\xba\x01\x15\x16\n\x0c\n\x02\x04\x14\x12\
    \x06\xbd\x01\0\xbf\x01\x01\n\x0b\n\x03\x04\x14\x01\x12\x04\xbd\x01\x08%\
    \n\x0c\n\x04\x04\x14\x02\0\x12\x04\xbe\x01\x04\x19\n\x0f\n\x05\x04\x14\
    \x02\0\x04\x12\x06\xbe\x01\x04\xbd\x01'\n\r\n\x05\x04\x14\x02\0\x05\x12\
    \x04\xbe\x01\x04\n\n\r\n\x05\x04\x14\x02\0\x01\x12\x04\xbe\x01\x0b\x14\n\
    \r\n\x05\x04\x14\x02\0\x03\x12\x04\xbe\x01\x17\x18\n\x0c\n\x02\x04\x15\
    \x12\x06\xc1\x01\0\xc2\x01\x01\n\x0b\n\x03\x04\x15\x01\x12\x04\xc1\x01\
    \x08&\n\x0c\n\x02\x04\x16\x12\x06\xc4\x01\0\xc8\x01\x01\n\x0b\n\x03\x04\
    \x16\x01\x12\x04\xc4\x01\x08\x1f\n\x0c\n\x04\x04\x16\x02\0\x12\x04\xc5\
    \x01\x04\x16\n\x0f\n\x05\x04\x16\x02\0\x04\x12\x06\xc5\x01\x04\xc4\x01!\
    \n\r\n\x05\x04\x16\x02\0\x06\x12\x04\xc5\x01\x04\n\n\r\n\x05\x04\x16\x02\
    \0\x01\x12\x04\xc5\x01\x0b\x11\n\r\n\x05\x04\x16\x02\0\x03\x12\x04\xc5\
    \x01\x14\x15\n\x0c\n\x04\x04\x16\x02\x01\x12\x04\xc6\x01\x04\x1b\n\x0f\n\
    \x05\x04\x16\x02\x01\x04\x12\x06\xc6\x01\x04\xc5\x01\x16\n\r\n\x05\x04\
    \x16\x02\x01\x05\x12\x04\xc6\x01\x04\n\n\r\n\x05\x04\x16\x02\x01\x01\x12\
    \x04\xc6\x01\x0b\x16\n\r\n\x05\x04\x16\x02\x01\x03\x12\x04\xc6\x01\x19\
    \x1a\n\x0c\n\x04\x04\x16\x02\x02\x12\x04\xc7\x01\x04\x1c\n\x0f\n\x05\x04\
    \x16\x02\x02\x04\x12\x06\xc7\x01\x04\xc6\x01\x1b\n\r\n\x05\x04\x16\x02\
    \x02\x05\x12\x04\xc7\x01\x04\n\n\r\n\x05\x04\x16\x02\x02\x01\x12\x04\xc7\
    \x01\x0b\x17\n\r\n\x05\x04\x16\x02\x02\x03\x12\x04\xc7\x01\x1a\x1b\n\x0c\
    \n\x02\x04\x17\x12\x06\xca\x01\0\xcb\x01\x01\n\x0b\n\x03\x04\x17\x01\x12\
    \x04\xca\x01\x08\x20\n\x0c\n\x02\x04\x18\x12\x06\xcd\x01\0\xd0\x01\x01\n\
    \x0b\n\x03\x04\x18\x01\x12\x04\xcd\x01\x08\x10\n\x0c\n\x04\x04\x18\x02\0\
    \x12\x04\xce\x01\x04\x14\n\x0f\n\x05\x04\x18\x02\0\x04\x12\x06\xce\x01\
    \x04\xcd\x01\x12\n\r\n\x05\x04\x18\x02\0\x05\x12\x04\xce\x01\x04\n\n\r\n\
    \x05\x04\x18\x02\0\x01\x12\x04\xce\x01\x0b\x0f\n\r\n\x05\x04\x18\x02\0\
    \x03\x12\x04\xce\x01\x12\x13\n\x0c\n\x04\x04\x18\x02\x01\x12\x04\xcf\x01\
    \x04\x15\n\x0f\n\x05\x04\x18\x02\x01\x04\x12\x06\xcf\x01\x04\xce\x01\x14\
    \n\r\n\x05\x04\x18\x02\x01\x05\x12\x04\xcf\x01\x04\n\n\r\n\x05\x04\x18\
    \x02\x01\x01\x12\x04\xcf\x01\x0b\x10\n\r\n\x05\x04\x18\x02\x01\x03\x12\
    \x04\xcf\x01\x13\x14\n\x0c\n\x02\x04\x19\x12\x06\xd2\x01\0\xd4\x01\x01\n\
    \x0b\n\x03\x04\x19\x01\x12\x04\xd2\x01\x08\"\n\x0c\n\x04\x04\x19\x02\0\
    \x12\x04\xd3\x01\x04\x19\n\x0f\n\x05\x04\x19\x02\0\x04\x12\x06\xd3\x01\
    \x04\xd2\x01$\n\r\n\x05\x04\x19\x02\0\x05\x12\x04\xd3\x01\x04\n\n\r\n\
    \x05\x04\x19\x02\0\x01\x12\x04\xd3\x01\x0b\x14\n\r\n\x05\x04\x19\x02\0\
    \x03\x12\x04\xd3\x01\x17\x18\n\x0c\n\x02\x04\x1a\x12\x06\xd6\x01\0\xd8\
    \x01\x01\n\x0b\n\x03\x04\x1a\x01\x12\x04\xd6\x01\x08#\n\x0c\n\x04\x04\
    \x1a\x02\0\x12\x04\xd7\x01\x04\x20\n\r\n\x05\x04\x1a\x02\0\x04\x12\x04\
    \xd7\x01\x04\x0c\n\r\n\x05\x04\x1a\x02\0\x06\x12\x04\xd7\x01\r\x15\n\r\n\
    \x05\x04\x1a\x02\0\x01\x12\x04\xd7\x01\x16\x1b\n\r\n\x05\x04\x1a\x02\0\
    \x03\x12\x04\xd7\x01\x1e\x1fb\x06proto3\
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
