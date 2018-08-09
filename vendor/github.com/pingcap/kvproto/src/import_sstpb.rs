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
pub struct Range {
    // message fields
    pub start: ::std::vec::Vec<u8>,
    pub end: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for Range {}

impl Range {
    pub fn new() -> Range {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static Range {
        static mut instance: ::protobuf::lazy::Lazy<Range> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const Range,
        };
        unsafe {
            instance.get(Range::new)
        }
    }

    // bytes start = 1;

    pub fn clear_start(&mut self) {
        self.start.clear();
    }

    // Param is passed by value, moved
    pub fn set_start(&mut self, v: ::std::vec::Vec<u8>) {
        self.start = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_start(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start
    }

    // Take field
    pub fn take_start(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.start, ::std::vec::Vec::new())
    }

    pub fn get_start(&self) -> &[u8] {
        &self.start
    }

    fn get_start_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.start
    }

    fn mut_start_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start
    }

    // bytes end = 2;

    pub fn clear_end(&mut self) {
        self.end.clear();
    }

    // Param is passed by value, moved
    pub fn set_end(&mut self, v: ::std::vec::Vec<u8>) {
        self.end = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_end(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.end
    }

    // Take field
    pub fn take_end(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.end, ::std::vec::Vec::new())
    }

    pub fn get_end(&self) -> &[u8] {
        &self.end
    }

    fn get_end_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.end
    }

    fn mut_end_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.end
    }
}

impl ::protobuf::Message for Range {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.start)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.end)?;
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
        if !self.start.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.start);
        }
        if !self.end.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.end);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.start.is_empty() {
            os.write_bytes(1, &self.start)?;
        }
        if !self.end.is_empty() {
            os.write_bytes(2, &self.end)?;
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

impl ::protobuf::MessageStatic for Range {
    fn new() -> Range {
        Range::new()
    }

    fn descriptor_static(_: ::std::option::Option<Range>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "start",
                    Range::get_start_for_reflect,
                    Range::mut_start_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "end",
                    Range::get_end_for_reflect,
                    Range::mut_end_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<Range>(
                    "Range",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for Range {
    fn clear(&mut self) {
        self.clear_start();
        self.clear_end();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for Range {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for Range {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct SSTMeta {
    // message fields
    pub uuid: ::std::vec::Vec<u8>,
    pub range: ::protobuf::SingularPtrField<Range>,
    pub crc32: u32,
    pub length: u64,
    pub cf_name: ::std::string::String,
    pub region_id: u64,
    pub region_epoch: ::protobuf::SingularPtrField<super::metapb::RegionEpoch>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for SSTMeta {}

impl SSTMeta {
    pub fn new() -> SSTMeta {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static SSTMeta {
        static mut instance: ::protobuf::lazy::Lazy<SSTMeta> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const SSTMeta,
        };
        unsafe {
            instance.get(SSTMeta::new)
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

    // .import_sstpb.Range range = 2;

    pub fn clear_range(&mut self) {
        self.range.clear();
    }

    pub fn has_range(&self) -> bool {
        self.range.is_some()
    }

    // Param is passed by value, moved
    pub fn set_range(&mut self, v: Range) {
        self.range = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_range(&mut self) -> &mut Range {
        if self.range.is_none() {
            self.range.set_default();
        }
        self.range.as_mut().unwrap()
    }

    // Take field
    pub fn take_range(&mut self) -> Range {
        self.range.take().unwrap_or_else(|| Range::new())
    }

    pub fn get_range(&self) -> &Range {
        self.range.as_ref().unwrap_or_else(|| Range::default_instance())
    }

    fn get_range_for_reflect(&self) -> &::protobuf::SingularPtrField<Range> {
        &self.range
    }

    fn mut_range_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Range> {
        &mut self.range
    }

    // uint32 crc32 = 3;

    pub fn clear_crc32(&mut self) {
        self.crc32 = 0;
    }

    // Param is passed by value, moved
    pub fn set_crc32(&mut self, v: u32) {
        self.crc32 = v;
    }

    pub fn get_crc32(&self) -> u32 {
        self.crc32
    }

    fn get_crc32_for_reflect(&self) -> &u32 {
        &self.crc32
    }

    fn mut_crc32_for_reflect(&mut self) -> &mut u32 {
        &mut self.crc32
    }

    // uint64 length = 4;

    pub fn clear_length(&mut self) {
        self.length = 0;
    }

    // Param is passed by value, moved
    pub fn set_length(&mut self, v: u64) {
        self.length = v;
    }

    pub fn get_length(&self) -> u64 {
        self.length
    }

    fn get_length_for_reflect(&self) -> &u64 {
        &self.length
    }

    fn mut_length_for_reflect(&mut self) -> &mut u64 {
        &mut self.length
    }

    // string cf_name = 5;

    pub fn clear_cf_name(&mut self) {
        self.cf_name.clear();
    }

    // Param is passed by value, moved
    pub fn set_cf_name(&mut self, v: ::std::string::String) {
        self.cf_name = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_cf_name(&mut self) -> &mut ::std::string::String {
        &mut self.cf_name
    }

    // Take field
    pub fn take_cf_name(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.cf_name, ::std::string::String::new())
    }

    pub fn get_cf_name(&self) -> &str {
        &self.cf_name
    }

    fn get_cf_name_for_reflect(&self) -> &::std::string::String {
        &self.cf_name
    }

    fn mut_cf_name_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.cf_name
    }

    // uint64 region_id = 6;

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

    // .metapb.RegionEpoch region_epoch = 7;

    pub fn clear_region_epoch(&mut self) {
        self.region_epoch.clear();
    }

    pub fn has_region_epoch(&self) -> bool {
        self.region_epoch.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_epoch(&mut self, v: super::metapb::RegionEpoch) {
        self.region_epoch = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_epoch(&mut self) -> &mut super::metapb::RegionEpoch {
        if self.region_epoch.is_none() {
            self.region_epoch.set_default();
        }
        self.region_epoch.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_epoch(&mut self) -> super::metapb::RegionEpoch {
        self.region_epoch.take().unwrap_or_else(|| super::metapb::RegionEpoch::new())
    }

    pub fn get_region_epoch(&self) -> &super::metapb::RegionEpoch {
        self.region_epoch.as_ref().unwrap_or_else(|| super::metapb::RegionEpoch::default_instance())
    }

    fn get_region_epoch_for_reflect(&self) -> &::protobuf::SingularPtrField<super::metapb::RegionEpoch> {
        &self.region_epoch
    }

    fn mut_region_epoch_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::metapb::RegionEpoch> {
        &mut self.region_epoch
    }
}

impl ::protobuf::Message for SSTMeta {
    fn is_initialized(&self) -> bool {
        for v in &self.range {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.region_epoch {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.uuid)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.range)?;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint32()?;
                    self.crc32 = tmp;
                },
                4 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.length = tmp;
                },
                5 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf_name)?;
                },
                6 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.region_id = tmp;
                },
                7 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_epoch)?;
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
        if let Some(ref v) = self.range.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if self.crc32 != 0 {
            my_size += ::protobuf::rt::value_size(3, self.crc32, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.length != 0 {
            my_size += ::protobuf::rt::value_size(4, self.length, ::protobuf::wire_format::WireTypeVarint);
        }
        if !self.cf_name.is_empty() {
            my_size += ::protobuf::rt::string_size(5, &self.cf_name);
        }
        if self.region_id != 0 {
            my_size += ::protobuf::rt::value_size(6, self.region_id, ::protobuf::wire_format::WireTypeVarint);
        }
        if let Some(ref v) = self.region_epoch.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.uuid.is_empty() {
            os.write_bytes(1, &self.uuid)?;
        }
        if let Some(ref v) = self.range.as_ref() {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if self.crc32 != 0 {
            os.write_uint32(3, self.crc32)?;
        }
        if self.length != 0 {
            os.write_uint64(4, self.length)?;
        }
        if !self.cf_name.is_empty() {
            os.write_string(5, &self.cf_name)?;
        }
        if self.region_id != 0 {
            os.write_uint64(6, self.region_id)?;
        }
        if let Some(ref v) = self.region_epoch.as_ref() {
            os.write_tag(7, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for SSTMeta {
    fn new() -> SSTMeta {
        SSTMeta::new()
    }

    fn descriptor_static(_: ::std::option::Option<SSTMeta>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "uuid",
                    SSTMeta::get_uuid_for_reflect,
                    SSTMeta::mut_uuid_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Range>>(
                    "range",
                    SSTMeta::get_range_for_reflect,
                    SSTMeta::mut_range_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint32>(
                    "crc32",
                    SSTMeta::get_crc32_for_reflect,
                    SSTMeta::mut_crc32_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "length",
                    SSTMeta::get_length_for_reflect,
                    SSTMeta::mut_length_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf_name",
                    SSTMeta::get_cf_name_for_reflect,
                    SSTMeta::mut_cf_name_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "region_id",
                    SSTMeta::get_region_id_for_reflect,
                    SSTMeta::mut_region_id_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::metapb::RegionEpoch>>(
                    "region_epoch",
                    SSTMeta::get_region_epoch_for_reflect,
                    SSTMeta::mut_region_epoch_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<SSTMeta>(
                    "SSTMeta",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for SSTMeta {
    fn clear(&mut self) {
        self.clear_uuid();
        self.clear_range();
        self.clear_crc32();
        self.clear_length();
        self.clear_cf_name();
        self.clear_region_id();
        self.clear_region_epoch();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for SSTMeta {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for SSTMeta {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct UploadRequest {
    // message oneof groups
    chunk: ::std::option::Option<UploadRequest_oneof_chunk>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for UploadRequest {}

#[derive(Clone,PartialEq)]
pub enum UploadRequest_oneof_chunk {
    meta(SSTMeta),
    data(::std::vec::Vec<u8>),
}

impl UploadRequest {
    pub fn new() -> UploadRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static UploadRequest {
        static mut instance: ::protobuf::lazy::Lazy<UploadRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const UploadRequest,
        };
        unsafe {
            instance.get(UploadRequest::new)
        }
    }

    // .import_sstpb.SSTMeta meta = 1;

    pub fn clear_meta(&mut self) {
        self.chunk = ::std::option::Option::None;
    }

    pub fn has_meta(&self) -> bool {
        match self.chunk {
            ::std::option::Option::Some(UploadRequest_oneof_chunk::meta(..)) => true,
            _ => false,
        }
    }

    // Param is passed by value, moved
    pub fn set_meta(&mut self, v: SSTMeta) {
        self.chunk = ::std::option::Option::Some(UploadRequest_oneof_chunk::meta(v))
    }

    // Mutable pointer to the field.
    pub fn mut_meta(&mut self) -> &mut SSTMeta {
        if let ::std::option::Option::Some(UploadRequest_oneof_chunk::meta(_)) = self.chunk {
        } else {
            self.chunk = ::std::option::Option::Some(UploadRequest_oneof_chunk::meta(SSTMeta::new()));
        }
        match self.chunk {
            ::std::option::Option::Some(UploadRequest_oneof_chunk::meta(ref mut v)) => v,
            _ => panic!(),
        }
    }

    // Take field
    pub fn take_meta(&mut self) -> SSTMeta {
        if self.has_meta() {
            match self.chunk.take() {
                ::std::option::Option::Some(UploadRequest_oneof_chunk::meta(v)) => v,
                _ => panic!(),
            }
        } else {
            SSTMeta::new()
        }
    }

    pub fn get_meta(&self) -> &SSTMeta {
        match self.chunk {
            ::std::option::Option::Some(UploadRequest_oneof_chunk::meta(ref v)) => v,
            _ => SSTMeta::default_instance(),
        }
    }

    // bytes data = 2;

    pub fn clear_data(&mut self) {
        self.chunk = ::std::option::Option::None;
    }

    pub fn has_data(&self) -> bool {
        match self.chunk {
            ::std::option::Option::Some(UploadRequest_oneof_chunk::data(..)) => true,
            _ => false,
        }
    }

    // Param is passed by value, moved
    pub fn set_data(&mut self, v: ::std::vec::Vec<u8>) {
        self.chunk = ::std::option::Option::Some(UploadRequest_oneof_chunk::data(v))
    }

    // Mutable pointer to the field.
    pub fn mut_data(&mut self) -> &mut ::std::vec::Vec<u8> {
        if let ::std::option::Option::Some(UploadRequest_oneof_chunk::data(_)) = self.chunk {
        } else {
            self.chunk = ::std::option::Option::Some(UploadRequest_oneof_chunk::data(::std::vec::Vec::new()));
        }
        match self.chunk {
            ::std::option::Option::Some(UploadRequest_oneof_chunk::data(ref mut v)) => v,
            _ => panic!(),
        }
    }

    // Take field
    pub fn take_data(&mut self) -> ::std::vec::Vec<u8> {
        if self.has_data() {
            match self.chunk.take() {
                ::std::option::Option::Some(UploadRequest_oneof_chunk::data(v)) => v,
                _ => panic!(),
            }
        } else {
            ::std::vec::Vec::new()
        }
    }

    pub fn get_data(&self) -> &[u8] {
        match self.chunk {
            ::std::option::Option::Some(UploadRequest_oneof_chunk::data(ref v)) => v,
            _ => &[],
        }
    }
}

impl ::protobuf::Message for UploadRequest {
    fn is_initialized(&self) -> bool {
        if let Some(UploadRequest_oneof_chunk::meta(ref v)) = self.chunk {
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
                    self.chunk = ::std::option::Option::Some(UploadRequest_oneof_chunk::meta(is.read_message()?));
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeLengthDelimited {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    self.chunk = ::std::option::Option::Some(UploadRequest_oneof_chunk::data(is.read_bytes()?));
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
                &UploadRequest_oneof_chunk::meta(ref v) => {
                    let len = v.compute_size();
                    my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
                },
                &UploadRequest_oneof_chunk::data(ref v) => {
                    my_size += ::protobuf::rt::bytes_size(2, &v);
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
                &UploadRequest_oneof_chunk::meta(ref v) => {
                    os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
                    os.write_raw_varint32(v.get_cached_size())?;
                    v.write_to_with_cached_sizes(os)?;
                },
                &UploadRequest_oneof_chunk::data(ref v) => {
                    os.write_bytes(2, v)?;
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

impl ::protobuf::MessageStatic for UploadRequest {
    fn new() -> UploadRequest {
        UploadRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<UploadRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_message_accessor::<_, SSTMeta>(
                    "meta",
                    UploadRequest::has_meta,
                    UploadRequest::get_meta,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_bytes_accessor::<_>(
                    "data",
                    UploadRequest::has_data,
                    UploadRequest::get_data,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<UploadRequest>(
                    "UploadRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for UploadRequest {
    fn clear(&mut self) {
        self.clear_meta();
        self.clear_data();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for UploadRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for UploadRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct UploadResponse {
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for UploadResponse {}

impl UploadResponse {
    pub fn new() -> UploadResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static UploadResponse {
        static mut instance: ::protobuf::lazy::Lazy<UploadResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const UploadResponse,
        };
        unsafe {
            instance.get(UploadResponse::new)
        }
    }
}

impl ::protobuf::Message for UploadResponse {
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

impl ::protobuf::MessageStatic for UploadResponse {
    fn new() -> UploadResponse {
        UploadResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<UploadResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let fields = ::std::vec::Vec::new();
                ::protobuf::reflect::MessageDescriptor::new::<UploadResponse>(
                    "UploadResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for UploadResponse {
    fn clear(&mut self) {
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for UploadResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for UploadResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct IngestRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<super::kvrpcpb::Context>,
    pub sst: ::protobuf::SingularPtrField<SSTMeta>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for IngestRequest {}

impl IngestRequest {
    pub fn new() -> IngestRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static IngestRequest {
        static mut instance: ::protobuf::lazy::Lazy<IngestRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const IngestRequest,
        };
        unsafe {
            instance.get(IngestRequest::new)
        }
    }

    // .kvrpcpb.Context context = 1;

    pub fn clear_context(&mut self) {
        self.context.clear();
    }

    pub fn has_context(&self) -> bool {
        self.context.is_some()
    }

    // Param is passed by value, moved
    pub fn set_context(&mut self, v: super::kvrpcpb::Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut super::kvrpcpb::Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> super::kvrpcpb::Context {
        self.context.take().unwrap_or_else(|| super::kvrpcpb::Context::new())
    }

    pub fn get_context(&self) -> &super::kvrpcpb::Context {
        self.context.as_ref().unwrap_or_else(|| super::kvrpcpb::Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<super::kvrpcpb::Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::kvrpcpb::Context> {
        &mut self.context
    }

    // .import_sstpb.SSTMeta sst = 2;

    pub fn clear_sst(&mut self) {
        self.sst.clear();
    }

    pub fn has_sst(&self) -> bool {
        self.sst.is_some()
    }

    // Param is passed by value, moved
    pub fn set_sst(&mut self, v: SSTMeta) {
        self.sst = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_sst(&mut self) -> &mut SSTMeta {
        if self.sst.is_none() {
            self.sst.set_default();
        }
        self.sst.as_mut().unwrap()
    }

    // Take field
    pub fn take_sst(&mut self) -> SSTMeta {
        self.sst.take().unwrap_or_else(|| SSTMeta::new())
    }

    pub fn get_sst(&self) -> &SSTMeta {
        self.sst.as_ref().unwrap_or_else(|| SSTMeta::default_instance())
    }

    fn get_sst_for_reflect(&self) -> &::protobuf::SingularPtrField<SSTMeta> {
        &self.sst
    }

    fn mut_sst_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<SSTMeta> {
        &mut self.sst
    }
}

impl ::protobuf::Message for IngestRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.sst {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.context)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.sst)?;
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
        if let Some(ref v) = self.context.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.sst.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.context.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.sst.as_ref() {
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

impl ::protobuf::MessageStatic for IngestRequest {
    fn new() -> IngestRequest {
        IngestRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<IngestRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::kvrpcpb::Context>>(
                    "context",
                    IngestRequest::get_context_for_reflect,
                    IngestRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<SSTMeta>>(
                    "sst",
                    IngestRequest::get_sst_for_reflect,
                    IngestRequest::mut_sst_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<IngestRequest>(
                    "IngestRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for IngestRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_sst();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for IngestRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for IngestRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct IngestResponse {
    // message fields
    pub error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for IngestResponse {}

impl IngestResponse {
    pub fn new() -> IngestResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static IngestResponse {
        static mut instance: ::protobuf::lazy::Lazy<IngestResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const IngestResponse,
        };
        unsafe {
            instance.get(IngestResponse::new)
        }
    }

    // .errorpb.Error error = 1;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: super::errorpb::Error) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut super::errorpb::Error {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> super::errorpb::Error {
        self.error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_error(&self) -> &super::errorpb::Error {
        self.error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.error
    }
}

impl ::protobuf::Message for IngestResponse {
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

impl ::protobuf::MessageStatic for IngestResponse {
    fn new() -> IngestResponse {
        IngestResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<IngestResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "error",
                    IngestResponse::get_error_for_reflect,
                    IngestResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<IngestResponse>(
                    "IngestResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for IngestResponse {
    fn clear(&mut self) {
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for IngestResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for IngestResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

static file_descriptor_proto_data: &'static [u8] = b"\
    \n\x12import_sstpb.proto\x12\x0cimport_sstpb\x1a\x0cmetapb.proto\x1a\rer\
    rorpb.proto\x1a\rkvrpcpb.proto\x1a\x14gogoproto/gogo.proto\"/\n\x05Range\
    \x12\x14\n\x05start\x18\x01\x20\x01(\x0cR\x05start\x12\x10\n\x03end\x18\
    \x02\x20\x01(\x0cR\x03end\"\xe4\x01\n\x07SSTMeta\x12\x12\n\x04uuid\x18\
    \x01\x20\x01(\x0cR\x04uuid\x12)\n\x05range\x18\x02\x20\x01(\x0b2\x13.imp\
    ort_sstpb.RangeR\x05range\x12\x14\n\x05crc32\x18\x03\x20\x01(\rR\x05crc3\
    2\x12\x16\n\x06length\x18\x04\x20\x01(\x04R\x06length\x12\x17\n\x07cf_na\
    me\x18\x05\x20\x01(\tR\x06cfName\x12\x1b\n\tregion_id\x18\x06\x20\x01(\
    \x04R\x08regionId\x126\n\x0cregion_epoch\x18\x07\x20\x01(\x0b2\x13.metap\
    b.RegionEpochR\x0bregionEpoch\"[\n\rUploadRequest\x12+\n\x04meta\x18\x01\
    \x20\x01(\x0b2\x15.import_sstpb.SSTMetaH\0R\x04meta\x12\x14\n\x04data\
    \x18\x02\x20\x01(\x0cH\0R\x04dataB\x07\n\x05chunk\"\x10\n\x0eUploadRespo\
    nse\"d\n\rIngestRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrp\
    cpb.ContextR\x07context\x12'\n\x03sst\x18\x02\x20\x01(\x0b2\x15.import_s\
    stpb.SSTMetaR\x03sst\"6\n\x0eIngestResponse\x12$\n\x05error\x18\x01\x20\
    \x01(\x0b2\x0e.errorpb.ErrorR\x05error2\x9b\x01\n\tImportSST\x12G\n\x06U\
    pload\x12\x1b.import_sstpb.UploadRequest\x1a\x1c.import_sstpb.UploadResp\
    onse\"\0(\x01\x12E\n\x06Ingest\x12\x1b.import_sstpb.IngestRequest\x1a\
    \x1c.import_sstpb.IngestResponse\"\0B&\n\x18com.pingcap.tikv.kvproto\xd0\
    \xe2\x1e\x01\xc8\xe2\x1e\x01\xe0\xe2\x1e\x01J\x95\x15\n\x06\x12\x04\0\0E\
    \x01\n\x08\n\x01\x0c\x12\x03\0\0\x12\n\x08\n\x01\x02\x12\x03\x02\x08\x14\
    \n\t\n\x02\x03\0\x12\x03\x04\x07\x15\n\t\n\x02\x03\x01\x12\x03\x05\x07\
    \x16\n\t\n\x02\x03\x02\x12\x03\x06\x07\x16\n\t\n\x02\x03\x03\x12\x03\x07\
    \x07\x1d\n\x08\n\x01\x08\x12\x03\t\0$\n\x0b\n\x04\x08\xe7\x07\0\x12\x03\
    \t\0$\n\x0c\n\x05\x08\xe7\x07\0\x02\x12\x03\t\x07\x1c\n\r\n\x06\x08\xe7\
    \x07\0\x02\0\x12\x03\t\x07\x1c\n\x0e\n\x07\x08\xe7\x07\0\x02\0\x01\x12\
    \x03\t\x08\x1b\n\x0c\n\x05\x08\xe7\x07\0\x03\x12\x03\t\x1f#\n\x08\n\x01\
    \x08\x12\x03\n\0(\n\x0b\n\x04\x08\xe7\x07\x01\x12\x03\n\0(\n\x0c\n\x05\
    \x08\xe7\x07\x01\x02\x12\x03\n\x07\x20\n\r\n\x06\x08\xe7\x07\x01\x02\0\
    \x12\x03\n\x07\x20\n\x0e\n\x07\x08\xe7\x07\x01\x02\0\x01\x12\x03\n\x08\
    \x1f\n\x0c\n\x05\x08\xe7\x07\x01\x03\x12\x03\n#'\n\x08\n\x01\x08\x12\x03\
    \x0b\0*\n\x0b\n\x04\x08\xe7\x07\x02\x12\x03\x0b\0*\n\x0c\n\x05\x08\xe7\
    \x07\x02\x02\x12\x03\x0b\x07\"\n\r\n\x06\x08\xe7\x07\x02\x02\0\x12\x03\
    \x0b\x07\"\n\x0e\n\x07\x08\xe7\x07\x02\x02\0\x01\x12\x03\x0b\x08!\n\x0c\
    \n\x05\x08\xe7\x07\x02\x03\x12\x03\x0b%)\n\x08\n\x01\x08\x12\x03\r\01\n\
    \x0b\n\x04\x08\xe7\x07\x03\x12\x03\r\01\n\x0c\n\x05\x08\xe7\x07\x03\x02\
    \x12\x03\r\x07\x13\n\r\n\x06\x08\xe7\x07\x03\x02\0\x12\x03\r\x07\x13\n\
    \x0e\n\x07\x08\xe7\x07\x03\x02\0\x01\x12\x03\r\x07\x13\n\x0c\n\x05\x08\
    \xe7\x07\x03\x07\x12\x03\r\x160\n\xc9\x07\n\x02\x06\0\x12\x04\x1e\0#\x01\
    \x1a\xbc\x07\x20ImportSST\x20provides\x20a\x20service\x20to\x20import\
    \x20a\x20generated\x20SST\x20file\x20to\x20a\x20region\x20in\x20TiKV.\n\
    \n\x20In\x20order\x20to\x20import\x20an\x20SST\x20file\x20to\x20a\x20reg\
    ion,\x20the\x20user\x20should:\n\x201.\x20Retrieve\x20the\x20meta\x20of\
    \x20the\x20region\x20according\x20to\x20the\x20SST\x20file's\x20range.\n\
    \x202.\x20Upload\x20the\x20SST\x20file\x20to\x20the\x20servers\x20where\
    \x20the\x20region's\x20peers\x20locate\x20in.\n\x203.\x20Issue\x20an\x20\
    ingest\x20request\x20to\x20the\x20region's\x20leader\x20with\x20the\x20S\
    ST\x20file's\x20metadata.\n\n\x20It's\x20the\x20user's\x20responsibility\
    \x20to\x20make\x20sure\x20that\x20the\x20SST\x20file\x20is\x20uploaded\
    \x20to\n\x20the\x20servers\x20where\x20the\x20region's\x20peers\x20locat\
    e\x20in,\x20before\x20issue\x20the\x20ingest\n\x20request\x20to\x20the\
    \x20region's\x20leader.\x20However,\x20the\x20region\x20can\x20be\x20sch\
    eduled\x20(so\x20the\n\x20location\x20of\x20the\x20region's\x20peers\x20\
    will\x20be\x20changed)\x20or\x20split/merged\x20(so\x20the\x20range\n\
    \x20of\x20the\x20region\x20will\x20be\x20changed),\x20after\x20the\x20SS\
    T\x20file\x20is\x20uploaded,\x20but\x20before\n\x20the\x20SST\x20file\
    \x20is\x20ingested.\x20So,\x20the\x20region's\x20epoch\x20is\x20provided\
    \x20in\x20the\x20SST\n\x20file's\x20metadata,\x20to\x20guarantee\x20that\
    \x20the\x20region's\x20epoch\x20must\x20be\x20the\x20same\n\x20between\
    \x20the\x20SST\x20file\x20is\x20uploaded\x20and\x20ingested\x20later.\n\
    \n\n\n\x03\x06\0\x01\x12\x03\x1e\x08\x11\n.\n\x04\x06\0\x02\0\x12\x03\
    \x20\x04@\x1a!\x20Upload\x20an\x20SST\x20file\x20to\x20a\x20server.\n\n\
    \x0c\n\x05\x06\0\x02\0\x01\x12\x03\x20\x08\x0e\n\x0c\n\x05\x06\0\x02\0\
    \x05\x12\x03\x20\x0f\x15\n\x0c\n\x05\x06\0\x02\0\x02\x12\x03\x20\x16#\n\
    \x0c\n\x05\x06\0\x02\0\x03\x12\x03\x20.<\n7\n\x04\x06\0\x02\x01\x12\x03\
    \"\x049\x1a*\x20Ingest\x20an\x20uploaded\x20SST\x20file\x20to\x20a\x20re\
    gion.\n\n\x0c\n\x05\x06\0\x02\x01\x01\x12\x03\"\x08\x0e\n\x0c\n\x05\x06\
    \0\x02\x01\x02\x12\x03\"\x0f\x1c\n\x0c\n\x05\x06\0\x02\x01\x03\x12\x03\"\
    '5\n\n\n\x02\x04\0\x12\x04%\0(\x01\n\n\n\x03\x04\0\x01\x12\x03%\x08\r\n\
    \x0b\n\x04\x04\0\x02\0\x12\x03&\x04\x14\n\r\n\x05\x04\0\x02\0\x04\x12\
    \x04&\x04%\x0f\n\x0c\n\x05\x04\0\x02\0\x05\x12\x03&\x04\t\n\x0c\n\x05\
    \x04\0\x02\0\x01\x12\x03&\n\x0f\n\x0c\n\x05\x04\0\x02\0\x03\x12\x03&\x12\
    \x13\n\x0b\n\x04\x04\0\x02\x01\x12\x03'\x04\x12\n\r\n\x05\x04\0\x02\x01\
    \x04\x12\x04'\x04&\x14\n\x0c\n\x05\x04\0\x02\x01\x05\x12\x03'\x04\t\n\
    \x0c\n\x05\x04\0\x02\x01\x01\x12\x03'\n\r\n\x0c\n\x05\x04\0\x02\x01\x03\
    \x12\x03'\x10\x11\n\n\n\x02\x04\x01\x12\x04*\02\x01\n\n\n\x03\x04\x01\
    \x01\x12\x03*\x08\x0f\n\x0b\n\x04\x04\x01\x02\0\x12\x03+\x04\x13\n\r\n\
    \x05\x04\x01\x02\0\x04\x12\x04+\x04*\x11\n\x0c\n\x05\x04\x01\x02\0\x05\
    \x12\x03+\x04\t\n\x0c\n\x05\x04\x01\x02\0\x01\x12\x03+\n\x0e\n\x0c\n\x05\
    \x04\x01\x02\0\x03\x12\x03+\x11\x12\n\x0b\n\x04\x04\x01\x02\x01\x12\x03,\
    \x04\x14\n\r\n\x05\x04\x01\x02\x01\x04\x12\x04,\x04+\x13\n\x0c\n\x05\x04\
    \x01\x02\x01\x06\x12\x03,\x04\t\n\x0c\n\x05\x04\x01\x02\x01\x01\x12\x03,\
    \n\x0f\n\x0c\n\x05\x04\x01\x02\x01\x03\x12\x03,\x12\x13\n\x0b\n\x04\x04\
    \x01\x02\x02\x12\x03-\x04\x15\n\r\n\x05\x04\x01\x02\x02\x04\x12\x04-\x04\
    ,\x14\n\x0c\n\x05\x04\x01\x02\x02\x05\x12\x03-\x04\n\n\x0c\n\x05\x04\x01\
    \x02\x02\x01\x12\x03-\x0b\x10\n\x0c\n\x05\x04\x01\x02\x02\x03\x12\x03-\
    \x13\x14\n\x0b\n\x04\x04\x01\x02\x03\x12\x03.\x04\x16\n\r\n\x05\x04\x01\
    \x02\x03\x04\x12\x04.\x04-\x15\n\x0c\n\x05\x04\x01\x02\x03\x05\x12\x03.\
    \x04\n\n\x0c\n\x05\x04\x01\x02\x03\x01\x12\x03.\x0b\x11\n\x0c\n\x05\x04\
    \x01\x02\x03\x03\x12\x03.\x14\x15\n\x0b\n\x04\x04\x01\x02\x04\x12\x03/\
    \x04\x17\n\r\n\x05\x04\x01\x02\x04\x04\x12\x04/\x04.\x16\n\x0c\n\x05\x04\
    \x01\x02\x04\x05\x12\x03/\x04\n\n\x0c\n\x05\x04\x01\x02\x04\x01\x12\x03/\
    \x0b\x12\n\x0c\n\x05\x04\x01\x02\x04\x03\x12\x03/\x15\x16\n\x0b\n\x04\
    \x04\x01\x02\x05\x12\x030\x04\x19\n\r\n\x05\x04\x01\x02\x05\x04\x12\x040\
    \x04/\x17\n\x0c\n\x05\x04\x01\x02\x05\x05\x12\x030\x04\n\n\x0c\n\x05\x04\
    \x01\x02\x05\x01\x12\x030\x0b\x14\n\x0c\n\x05\x04\x01\x02\x05\x03\x12\
    \x030\x17\x18\n\x0b\n\x04\x04\x01\x02\x06\x12\x031\x04(\n\r\n\x05\x04\
    \x01\x02\x06\x04\x12\x041\x040\x19\n\x0c\n\x05\x04\x01\x02\x06\x06\x12\
    \x031\x04\x16\n\x0c\n\x05\x04\x01\x02\x06\x01\x12\x031\x17#\n\x0c\n\x05\
    \x04\x01\x02\x06\x03\x12\x031&'\n\n\n\x02\x04\x02\x12\x044\09\x01\n\n\n\
    \x03\x04\x02\x01\x12\x034\x08\x15\n\x0c\n\x04\x04\x02\x08\0\x12\x045\x04\
    8\x05\n\x0c\n\x05\x04\x02\x08\0\x01\x12\x035\n\x0f\n\x0b\n\x04\x04\x02\
    \x02\0\x12\x036\x08\x19\n\x0c\n\x05\x04\x02\x02\0\x06\x12\x036\x08\x0f\n\
    \x0c\n\x05\x04\x02\x02\0\x01\x12\x036\x10\x14\n\x0c\n\x05\x04\x02\x02\0\
    \x03\x12\x036\x17\x18\n\x0b\n\x04\x04\x02\x02\x01\x12\x037\x08\x17\n\x0c\
    \n\x05\x04\x02\x02\x01\x05\x12\x037\x08\r\n\x0c\n\x05\x04\x02\x02\x01\
    \x01\x12\x037\x0e\x12\n\x0c\n\x05\x04\x02\x02\x01\x03\x12\x037\x15\x16\n\
    \n\n\x02\x04\x03\x12\x04;\0<\x01\n\n\n\x03\x04\x03\x01\x12\x03;\x08\x16\
    \n\n\n\x02\x04\x04\x12\x04>\0A\x01\n\n\n\x03\x04\x04\x01\x12\x03>\x08\
    \x15\n\x0b\n\x04\x04\x04\x02\0\x12\x03?\x04\x20\n\r\n\x05\x04\x04\x02\0\
    \x04\x12\x04?\x04>\x17\n\x0c\n\x05\x04\x04\x02\0\x06\x12\x03?\x04\x13\n\
    \x0c\n\x05\x04\x04\x02\0\x01\x12\x03?\x14\x1b\n\x0c\n\x05\x04\x04\x02\0\
    \x03\x12\x03?\x1e\x1f\n\x0b\n\x04\x04\x04\x02\x01\x12\x03@\x04\x14\n\r\n\
    \x05\x04\x04\x02\x01\x04\x12\x04@\x04?\x20\n\x0c\n\x05\x04\x04\x02\x01\
    \x06\x12\x03@\x04\x0b\n\x0c\n\x05\x04\x04\x02\x01\x01\x12\x03@\x0c\x0f\n\
    \x0c\n\x05\x04\x04\x02\x01\x03\x12\x03@\x12\x13\n\n\n\x02\x04\x05\x12\
    \x04C\0E\x01\n\n\n\x03\x04\x05\x01\x12\x03C\x08\x16\n\x0b\n\x04\x04\x05\
    \x02\0\x12\x03D\x04\x1c\n\r\n\x05\x04\x05\x02\0\x04\x12\x04D\x04C\x18\n\
    \x0c\n\x05\x04\x05\x02\0\x06\x12\x03D\x04\x11\n\x0c\n\x05\x04\x05\x02\0\
    \x01\x12\x03D\x12\x17\n\x0c\n\x05\x04\x05\x02\0\x03\x12\x03D\x1a\x1bb\
    \x06proto3\
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
