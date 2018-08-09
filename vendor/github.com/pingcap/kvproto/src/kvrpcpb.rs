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
pub struct LockInfo {
    // message fields
    pub primary_lock: ::std::vec::Vec<u8>,
    pub lock_version: u64,
    pub key: ::std::vec::Vec<u8>,
    pub lock_ttl: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for LockInfo {}

impl LockInfo {
    pub fn new() -> LockInfo {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static LockInfo {
        static mut instance: ::protobuf::lazy::Lazy<LockInfo> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const LockInfo,
        };
        unsafe {
            instance.get(LockInfo::new)
        }
    }

    // bytes primary_lock = 1;

    pub fn clear_primary_lock(&mut self) {
        self.primary_lock.clear();
    }

    // Param is passed by value, moved
    pub fn set_primary_lock(&mut self, v: ::std::vec::Vec<u8>) {
        self.primary_lock = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_primary_lock(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.primary_lock
    }

    // Take field
    pub fn take_primary_lock(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.primary_lock, ::std::vec::Vec::new())
    }

    pub fn get_primary_lock(&self) -> &[u8] {
        &self.primary_lock
    }

    fn get_primary_lock_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.primary_lock
    }

    fn mut_primary_lock_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.primary_lock
    }

    // uint64 lock_version = 2;

    pub fn clear_lock_version(&mut self) {
        self.lock_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_lock_version(&mut self, v: u64) {
        self.lock_version = v;
    }

    pub fn get_lock_version(&self) -> u64 {
        self.lock_version
    }

    fn get_lock_version_for_reflect(&self) -> &u64 {
        &self.lock_version
    }

    fn mut_lock_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.lock_version
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

    // uint64 lock_ttl = 4;

    pub fn clear_lock_ttl(&mut self) {
        self.lock_ttl = 0;
    }

    // Param is passed by value, moved
    pub fn set_lock_ttl(&mut self, v: u64) {
        self.lock_ttl = v;
    }

    pub fn get_lock_ttl(&self) -> u64 {
        self.lock_ttl
    }

    fn get_lock_ttl_for_reflect(&self) -> &u64 {
        &self.lock_ttl
    }

    fn mut_lock_ttl_for_reflect(&mut self) -> &mut u64 {
        &mut self.lock_ttl
    }
}

impl ::protobuf::Message for LockInfo {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.primary_lock)?;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.lock_version = tmp;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                4 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.lock_ttl = tmp;
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
        if !self.primary_lock.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.primary_lock);
        }
        if self.lock_version != 0 {
            my_size += ::protobuf::rt::value_size(2, self.lock_version, ::protobuf::wire_format::WireTypeVarint);
        }
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.key);
        }
        if self.lock_ttl != 0 {
            my_size += ::protobuf::rt::value_size(4, self.lock_ttl, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.primary_lock.is_empty() {
            os.write_bytes(1, &self.primary_lock)?;
        }
        if self.lock_version != 0 {
            os.write_uint64(2, self.lock_version)?;
        }
        if !self.key.is_empty() {
            os.write_bytes(3, &self.key)?;
        }
        if self.lock_ttl != 0 {
            os.write_uint64(4, self.lock_ttl)?;
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

impl ::protobuf::MessageStatic for LockInfo {
    fn new() -> LockInfo {
        LockInfo::new()
    }

    fn descriptor_static(_: ::std::option::Option<LockInfo>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "primary_lock",
                    LockInfo::get_primary_lock_for_reflect,
                    LockInfo::mut_primary_lock_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "lock_version",
                    LockInfo::get_lock_version_for_reflect,
                    LockInfo::mut_lock_version_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    LockInfo::get_key_for_reflect,
                    LockInfo::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "lock_ttl",
                    LockInfo::get_lock_ttl_for_reflect,
                    LockInfo::mut_lock_ttl_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<LockInfo>(
                    "LockInfo",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for LockInfo {
    fn clear(&mut self) {
        self.clear_primary_lock();
        self.clear_lock_version();
        self.clear_key();
        self.clear_lock_ttl();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for LockInfo {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for LockInfo {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct KeyError {
    // message fields
    pub locked: ::protobuf::SingularPtrField<LockInfo>,
    pub retryable: ::std::string::String,
    pub abort: ::std::string::String,
    pub conflict: ::protobuf::SingularPtrField<WriteConflict>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for KeyError {}

impl KeyError {
    pub fn new() -> KeyError {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static KeyError {
        static mut instance: ::protobuf::lazy::Lazy<KeyError> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const KeyError,
        };
        unsafe {
            instance.get(KeyError::new)
        }
    }

    // .kvrpcpb.LockInfo locked = 1;

    pub fn clear_locked(&mut self) {
        self.locked.clear();
    }

    pub fn has_locked(&self) -> bool {
        self.locked.is_some()
    }

    // Param is passed by value, moved
    pub fn set_locked(&mut self, v: LockInfo) {
        self.locked = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_locked(&mut self) -> &mut LockInfo {
        if self.locked.is_none() {
            self.locked.set_default();
        }
        self.locked.as_mut().unwrap()
    }

    // Take field
    pub fn take_locked(&mut self) -> LockInfo {
        self.locked.take().unwrap_or_else(|| LockInfo::new())
    }

    pub fn get_locked(&self) -> &LockInfo {
        self.locked.as_ref().unwrap_or_else(|| LockInfo::default_instance())
    }

    fn get_locked_for_reflect(&self) -> &::protobuf::SingularPtrField<LockInfo> {
        &self.locked
    }

    fn mut_locked_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<LockInfo> {
        &mut self.locked
    }

    // string retryable = 2;

    pub fn clear_retryable(&mut self) {
        self.retryable.clear();
    }

    // Param is passed by value, moved
    pub fn set_retryable(&mut self, v: ::std::string::String) {
        self.retryable = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_retryable(&mut self) -> &mut ::std::string::String {
        &mut self.retryable
    }

    // Take field
    pub fn take_retryable(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.retryable, ::std::string::String::new())
    }

    pub fn get_retryable(&self) -> &str {
        &self.retryable
    }

    fn get_retryable_for_reflect(&self) -> &::std::string::String {
        &self.retryable
    }

    fn mut_retryable_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.retryable
    }

    // string abort = 3;

    pub fn clear_abort(&mut self) {
        self.abort.clear();
    }

    // Param is passed by value, moved
    pub fn set_abort(&mut self, v: ::std::string::String) {
        self.abort = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_abort(&mut self) -> &mut ::std::string::String {
        &mut self.abort
    }

    // Take field
    pub fn take_abort(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.abort, ::std::string::String::new())
    }

    pub fn get_abort(&self) -> &str {
        &self.abort
    }

    fn get_abort_for_reflect(&self) -> &::std::string::String {
        &self.abort
    }

    fn mut_abort_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.abort
    }

    // .kvrpcpb.WriteConflict conflict = 4;

    pub fn clear_conflict(&mut self) {
        self.conflict.clear();
    }

    pub fn has_conflict(&self) -> bool {
        self.conflict.is_some()
    }

    // Param is passed by value, moved
    pub fn set_conflict(&mut self, v: WriteConflict) {
        self.conflict = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_conflict(&mut self) -> &mut WriteConflict {
        if self.conflict.is_none() {
            self.conflict.set_default();
        }
        self.conflict.as_mut().unwrap()
    }

    // Take field
    pub fn take_conflict(&mut self) -> WriteConflict {
        self.conflict.take().unwrap_or_else(|| WriteConflict::new())
    }

    pub fn get_conflict(&self) -> &WriteConflict {
        self.conflict.as_ref().unwrap_or_else(|| WriteConflict::default_instance())
    }

    fn get_conflict_for_reflect(&self) -> &::protobuf::SingularPtrField<WriteConflict> {
        &self.conflict
    }

    fn mut_conflict_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<WriteConflict> {
        &mut self.conflict
    }
}

impl ::protobuf::Message for KeyError {
    fn is_initialized(&self) -> bool {
        for v in &self.locked {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.conflict {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.locked)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.retryable)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.abort)?;
                },
                4 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.conflict)?;
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
        if let Some(ref v) = self.locked.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.retryable.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.retryable);
        }
        if !self.abort.is_empty() {
            my_size += ::protobuf::rt::string_size(3, &self.abort);
        }
        if let Some(ref v) = self.conflict.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.locked.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.retryable.is_empty() {
            os.write_string(2, &self.retryable)?;
        }
        if !self.abort.is_empty() {
            os.write_string(3, &self.abort)?;
        }
        if let Some(ref v) = self.conflict.as_ref() {
            os.write_tag(4, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for KeyError {
    fn new() -> KeyError {
        KeyError::new()
    }

    fn descriptor_static(_: ::std::option::Option<KeyError>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<LockInfo>>(
                    "locked",
                    KeyError::get_locked_for_reflect,
                    KeyError::mut_locked_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "retryable",
                    KeyError::get_retryable_for_reflect,
                    KeyError::mut_retryable_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "abort",
                    KeyError::get_abort_for_reflect,
                    KeyError::mut_abort_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<WriteConflict>>(
                    "conflict",
                    KeyError::get_conflict_for_reflect,
                    KeyError::mut_conflict_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<KeyError>(
                    "KeyError",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for KeyError {
    fn clear(&mut self) {
        self.clear_locked();
        self.clear_retryable();
        self.clear_abort();
        self.clear_conflict();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for KeyError {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for KeyError {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct WriteConflict {
    // message fields
    pub start_ts: u64,
    pub conflict_ts: u64,
    pub key: ::std::vec::Vec<u8>,
    pub primary: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for WriteConflict {}

impl WriteConflict {
    pub fn new() -> WriteConflict {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static WriteConflict {
        static mut instance: ::protobuf::lazy::Lazy<WriteConflict> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const WriteConflict,
        };
        unsafe {
            instance.get(WriteConflict::new)
        }
    }

    // uint64 start_ts = 1;

    pub fn clear_start_ts(&mut self) {
        self.start_ts = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_ts(&mut self, v: u64) {
        self.start_ts = v;
    }

    pub fn get_start_ts(&self) -> u64 {
        self.start_ts
    }

    fn get_start_ts_for_reflect(&self) -> &u64 {
        &self.start_ts
    }

    fn mut_start_ts_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_ts
    }

    // uint64 conflict_ts = 2;

    pub fn clear_conflict_ts(&mut self) {
        self.conflict_ts = 0;
    }

    // Param is passed by value, moved
    pub fn set_conflict_ts(&mut self, v: u64) {
        self.conflict_ts = v;
    }

    pub fn get_conflict_ts(&self) -> u64 {
        self.conflict_ts
    }

    fn get_conflict_ts_for_reflect(&self) -> &u64 {
        &self.conflict_ts
    }

    fn mut_conflict_ts_for_reflect(&mut self) -> &mut u64 {
        &mut self.conflict_ts
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

    // bytes primary = 4;

    pub fn clear_primary(&mut self) {
        self.primary.clear();
    }

    // Param is passed by value, moved
    pub fn set_primary(&mut self, v: ::std::vec::Vec<u8>) {
        self.primary = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_primary(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.primary
    }

    // Take field
    pub fn take_primary(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.primary, ::std::vec::Vec::new())
    }

    pub fn get_primary(&self) -> &[u8] {
        &self.primary
    }

    fn get_primary_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.primary
    }

    fn mut_primary_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.primary
    }
}

impl ::protobuf::Message for WriteConflict {
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
                    self.start_ts = tmp;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.conflict_ts = tmp;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                4 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.primary)?;
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
        if self.start_ts != 0 {
            my_size += ::protobuf::rt::value_size(1, self.start_ts, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.conflict_ts != 0 {
            my_size += ::protobuf::rt::value_size(2, self.conflict_ts, ::protobuf::wire_format::WireTypeVarint);
        }
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.key);
        }
        if !self.primary.is_empty() {
            my_size += ::protobuf::rt::bytes_size(4, &self.primary);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.start_ts != 0 {
            os.write_uint64(1, self.start_ts)?;
        }
        if self.conflict_ts != 0 {
            os.write_uint64(2, self.conflict_ts)?;
        }
        if !self.key.is_empty() {
            os.write_bytes(3, &self.key)?;
        }
        if !self.primary.is_empty() {
            os.write_bytes(4, &self.primary)?;
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

impl ::protobuf::MessageStatic for WriteConflict {
    fn new() -> WriteConflict {
        WriteConflict::new()
    }

    fn descriptor_static(_: ::std::option::Option<WriteConflict>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_ts",
                    WriteConflict::get_start_ts_for_reflect,
                    WriteConflict::mut_start_ts_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "conflict_ts",
                    WriteConflict::get_conflict_ts_for_reflect,
                    WriteConflict::mut_conflict_ts_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    WriteConflict::get_key_for_reflect,
                    WriteConflict::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "primary",
                    WriteConflict::get_primary_for_reflect,
                    WriteConflict::mut_primary_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<WriteConflict>(
                    "WriteConflict",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for WriteConflict {
    fn clear(&mut self) {
        self.clear_start_ts();
        self.clear_conflict_ts();
        self.clear_key();
        self.clear_primary();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for WriteConflict {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for WriteConflict {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct Context {
    // message fields
    pub region_id: u64,
    pub region_epoch: ::protobuf::SingularPtrField<super::metapb::RegionEpoch>,
    pub peer: ::protobuf::SingularPtrField<super::metapb::Peer>,
    pub term: u64,
    pub priority: CommandPri,
    pub isolation_level: IsolationLevel,
    pub not_fill_cache: bool,
    pub sync_log: bool,
    pub handle_time: bool,
    pub scan_detail: bool,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for Context {}

impl Context {
    pub fn new() -> Context {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static Context {
        static mut instance: ::protobuf::lazy::Lazy<Context> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const Context,
        };
        unsafe {
            instance.get(Context::new)
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

    // .metapb.RegionEpoch region_epoch = 2;

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

    // .metapb.Peer peer = 3;

    pub fn clear_peer(&mut self) {
        self.peer.clear();
    }

    pub fn has_peer(&self) -> bool {
        self.peer.is_some()
    }

    // Param is passed by value, moved
    pub fn set_peer(&mut self, v: super::metapb::Peer) {
        self.peer = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_peer(&mut self) -> &mut super::metapb::Peer {
        if self.peer.is_none() {
            self.peer.set_default();
        }
        self.peer.as_mut().unwrap()
    }

    // Take field
    pub fn take_peer(&mut self) -> super::metapb::Peer {
        self.peer.take().unwrap_or_else(|| super::metapb::Peer::new())
    }

    pub fn get_peer(&self) -> &super::metapb::Peer {
        self.peer.as_ref().unwrap_or_else(|| super::metapb::Peer::default_instance())
    }

    fn get_peer_for_reflect(&self) -> &::protobuf::SingularPtrField<super::metapb::Peer> {
        &self.peer
    }

    fn mut_peer_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::metapb::Peer> {
        &mut self.peer
    }

    // uint64 term = 5;

    pub fn clear_term(&mut self) {
        self.term = 0;
    }

    // Param is passed by value, moved
    pub fn set_term(&mut self, v: u64) {
        self.term = v;
    }

    pub fn get_term(&self) -> u64 {
        self.term
    }

    fn get_term_for_reflect(&self) -> &u64 {
        &self.term
    }

    fn mut_term_for_reflect(&mut self) -> &mut u64 {
        &mut self.term
    }

    // .kvrpcpb.CommandPri priority = 6;

    pub fn clear_priority(&mut self) {
        self.priority = CommandPri::Normal;
    }

    // Param is passed by value, moved
    pub fn set_priority(&mut self, v: CommandPri) {
        self.priority = v;
    }

    pub fn get_priority(&self) -> CommandPri {
        self.priority
    }

    fn get_priority_for_reflect(&self) -> &CommandPri {
        &self.priority
    }

    fn mut_priority_for_reflect(&mut self) -> &mut CommandPri {
        &mut self.priority
    }

    // .kvrpcpb.IsolationLevel isolation_level = 7;

    pub fn clear_isolation_level(&mut self) {
        self.isolation_level = IsolationLevel::SI;
    }

    // Param is passed by value, moved
    pub fn set_isolation_level(&mut self, v: IsolationLevel) {
        self.isolation_level = v;
    }

    pub fn get_isolation_level(&self) -> IsolationLevel {
        self.isolation_level
    }

    fn get_isolation_level_for_reflect(&self) -> &IsolationLevel {
        &self.isolation_level
    }

    fn mut_isolation_level_for_reflect(&mut self) -> &mut IsolationLevel {
        &mut self.isolation_level
    }

    // bool not_fill_cache = 8;

    pub fn clear_not_fill_cache(&mut self) {
        self.not_fill_cache = false;
    }

    // Param is passed by value, moved
    pub fn set_not_fill_cache(&mut self, v: bool) {
        self.not_fill_cache = v;
    }

    pub fn get_not_fill_cache(&self) -> bool {
        self.not_fill_cache
    }

    fn get_not_fill_cache_for_reflect(&self) -> &bool {
        &self.not_fill_cache
    }

    fn mut_not_fill_cache_for_reflect(&mut self) -> &mut bool {
        &mut self.not_fill_cache
    }

    // bool sync_log = 9;

    pub fn clear_sync_log(&mut self) {
        self.sync_log = false;
    }

    // Param is passed by value, moved
    pub fn set_sync_log(&mut self, v: bool) {
        self.sync_log = v;
    }

    pub fn get_sync_log(&self) -> bool {
        self.sync_log
    }

    fn get_sync_log_for_reflect(&self) -> &bool {
        &self.sync_log
    }

    fn mut_sync_log_for_reflect(&mut self) -> &mut bool {
        &mut self.sync_log
    }

    // bool handle_time = 10;

    pub fn clear_handle_time(&mut self) {
        self.handle_time = false;
    }

    // Param is passed by value, moved
    pub fn set_handle_time(&mut self, v: bool) {
        self.handle_time = v;
    }

    pub fn get_handle_time(&self) -> bool {
        self.handle_time
    }

    fn get_handle_time_for_reflect(&self) -> &bool {
        &self.handle_time
    }

    fn mut_handle_time_for_reflect(&mut self) -> &mut bool {
        &mut self.handle_time
    }

    // bool scan_detail = 11;

    pub fn clear_scan_detail(&mut self) {
        self.scan_detail = false;
    }

    // Param is passed by value, moved
    pub fn set_scan_detail(&mut self, v: bool) {
        self.scan_detail = v;
    }

    pub fn get_scan_detail(&self) -> bool {
        self.scan_detail
    }

    fn get_scan_detail_for_reflect(&self) -> &bool {
        &self.scan_detail
    }

    fn mut_scan_detail_for_reflect(&mut self) -> &mut bool {
        &mut self.scan_detail
    }
}

impl ::protobuf::Message for Context {
    fn is_initialized(&self) -> bool {
        for v in &self.region_epoch {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.peer {
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
                    self.region_id = tmp;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_epoch)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.peer)?;
                },
                5 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.term = tmp;
                },
                6 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_enum()?;
                    self.priority = tmp;
                },
                7 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_enum()?;
                    self.isolation_level = tmp;
                },
                8 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_bool()?;
                    self.not_fill_cache = tmp;
                },
                9 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_bool()?;
                    self.sync_log = tmp;
                },
                10 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_bool()?;
                    self.handle_time = tmp;
                },
                11 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_bool()?;
                    self.scan_detail = tmp;
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
        if let Some(ref v) = self.region_epoch.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.peer.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if self.term != 0 {
            my_size += ::protobuf::rt::value_size(5, self.term, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.priority != CommandPri::Normal {
            my_size += ::protobuf::rt::enum_size(6, self.priority);
        }
        if self.isolation_level != IsolationLevel::SI {
            my_size += ::protobuf::rt::enum_size(7, self.isolation_level);
        }
        if self.not_fill_cache != false {
            my_size += 2;
        }
        if self.sync_log != false {
            my_size += 2;
        }
        if self.handle_time != false {
            my_size += 2;
        }
        if self.scan_detail != false {
            my_size += 2;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.region_id != 0 {
            os.write_uint64(1, self.region_id)?;
        }
        if let Some(ref v) = self.region_epoch.as_ref() {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.peer.as_ref() {
            os.write_tag(3, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if self.term != 0 {
            os.write_uint64(5, self.term)?;
        }
        if self.priority != CommandPri::Normal {
            os.write_enum(6, self.priority.value())?;
        }
        if self.isolation_level != IsolationLevel::SI {
            os.write_enum(7, self.isolation_level.value())?;
        }
        if self.not_fill_cache != false {
            os.write_bool(8, self.not_fill_cache)?;
        }
        if self.sync_log != false {
            os.write_bool(9, self.sync_log)?;
        }
        if self.handle_time != false {
            os.write_bool(10, self.handle_time)?;
        }
        if self.scan_detail != false {
            os.write_bool(11, self.scan_detail)?;
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

impl ::protobuf::MessageStatic for Context {
    fn new() -> Context {
        Context::new()
    }

    fn descriptor_static(_: ::std::option::Option<Context>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "region_id",
                    Context::get_region_id_for_reflect,
                    Context::mut_region_id_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::metapb::RegionEpoch>>(
                    "region_epoch",
                    Context::get_region_epoch_for_reflect,
                    Context::mut_region_epoch_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::metapb::Peer>>(
                    "peer",
                    Context::get_peer_for_reflect,
                    Context::mut_peer_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "term",
                    Context::get_term_for_reflect,
                    Context::mut_term_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeEnum<CommandPri>>(
                    "priority",
                    Context::get_priority_for_reflect,
                    Context::mut_priority_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeEnum<IsolationLevel>>(
                    "isolation_level",
                    Context::get_isolation_level_for_reflect,
                    Context::mut_isolation_level_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "not_fill_cache",
                    Context::get_not_fill_cache_for_reflect,
                    Context::mut_not_fill_cache_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "sync_log",
                    Context::get_sync_log_for_reflect,
                    Context::mut_sync_log_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "handle_time",
                    Context::get_handle_time_for_reflect,
                    Context::mut_handle_time_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "scan_detail",
                    Context::get_scan_detail_for_reflect,
                    Context::mut_scan_detail_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<Context>(
                    "Context",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for Context {
    fn clear(&mut self) {
        self.clear_region_id();
        self.clear_region_epoch();
        self.clear_peer();
        self.clear_term();
        self.clear_priority();
        self.clear_isolation_level();
        self.clear_not_fill_cache();
        self.clear_sync_log();
        self.clear_handle_time();
        self.clear_scan_detail();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for Context {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for Context {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct HandleTime {
    // message fields
    pub wait_ms: i64,
    pub process_ms: i64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for HandleTime {}

impl HandleTime {
    pub fn new() -> HandleTime {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static HandleTime {
        static mut instance: ::protobuf::lazy::Lazy<HandleTime> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const HandleTime,
        };
        unsafe {
            instance.get(HandleTime::new)
        }
    }

    // int64 wait_ms = 1;

    pub fn clear_wait_ms(&mut self) {
        self.wait_ms = 0;
    }

    // Param is passed by value, moved
    pub fn set_wait_ms(&mut self, v: i64) {
        self.wait_ms = v;
    }

    pub fn get_wait_ms(&self) -> i64 {
        self.wait_ms
    }

    fn get_wait_ms_for_reflect(&self) -> &i64 {
        &self.wait_ms
    }

    fn mut_wait_ms_for_reflect(&mut self) -> &mut i64 {
        &mut self.wait_ms
    }

    // int64 process_ms = 2;

    pub fn clear_process_ms(&mut self) {
        self.process_ms = 0;
    }

    // Param is passed by value, moved
    pub fn set_process_ms(&mut self, v: i64) {
        self.process_ms = v;
    }

    pub fn get_process_ms(&self) -> i64 {
        self.process_ms
    }

    fn get_process_ms_for_reflect(&self) -> &i64 {
        &self.process_ms
    }

    fn mut_process_ms_for_reflect(&mut self) -> &mut i64 {
        &mut self.process_ms
    }
}

impl ::protobuf::Message for HandleTime {
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
                    let tmp = is.read_int64()?;
                    self.wait_ms = tmp;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_int64()?;
                    self.process_ms = tmp;
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
        if self.wait_ms != 0 {
            my_size += ::protobuf::rt::value_size(1, self.wait_ms, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.process_ms != 0 {
            my_size += ::protobuf::rt::value_size(2, self.process_ms, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.wait_ms != 0 {
            os.write_int64(1, self.wait_ms)?;
        }
        if self.process_ms != 0 {
            os.write_int64(2, self.process_ms)?;
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

impl ::protobuf::MessageStatic for HandleTime {
    fn new() -> HandleTime {
        HandleTime::new()
    }

    fn descriptor_static(_: ::std::option::Option<HandleTime>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeInt64>(
                    "wait_ms",
                    HandleTime::get_wait_ms_for_reflect,
                    HandleTime::mut_wait_ms_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeInt64>(
                    "process_ms",
                    HandleTime::get_process_ms_for_reflect,
                    HandleTime::mut_process_ms_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<HandleTime>(
                    "HandleTime",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for HandleTime {
    fn clear(&mut self) {
        self.clear_wait_ms();
        self.clear_process_ms();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for HandleTime {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for HandleTime {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ScanInfo {
    // message fields
    pub total: i64,
    pub processed: i64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ScanInfo {}

impl ScanInfo {
    pub fn new() -> ScanInfo {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ScanInfo {
        static mut instance: ::protobuf::lazy::Lazy<ScanInfo> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ScanInfo,
        };
        unsafe {
            instance.get(ScanInfo::new)
        }
    }

    // int64 total = 1;

    pub fn clear_total(&mut self) {
        self.total = 0;
    }

    // Param is passed by value, moved
    pub fn set_total(&mut self, v: i64) {
        self.total = v;
    }

    pub fn get_total(&self) -> i64 {
        self.total
    }

    fn get_total_for_reflect(&self) -> &i64 {
        &self.total
    }

    fn mut_total_for_reflect(&mut self) -> &mut i64 {
        &mut self.total
    }

    // int64 processed = 2;

    pub fn clear_processed(&mut self) {
        self.processed = 0;
    }

    // Param is passed by value, moved
    pub fn set_processed(&mut self, v: i64) {
        self.processed = v;
    }

    pub fn get_processed(&self) -> i64 {
        self.processed
    }

    fn get_processed_for_reflect(&self) -> &i64 {
        &self.processed
    }

    fn mut_processed_for_reflect(&mut self) -> &mut i64 {
        &mut self.processed
    }
}

impl ::protobuf::Message for ScanInfo {
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
                    let tmp = is.read_int64()?;
                    self.total = tmp;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_int64()?;
                    self.processed = tmp;
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
        if self.total != 0 {
            my_size += ::protobuf::rt::value_size(1, self.total, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.processed != 0 {
            my_size += ::protobuf::rt::value_size(2, self.processed, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.total != 0 {
            os.write_int64(1, self.total)?;
        }
        if self.processed != 0 {
            os.write_int64(2, self.processed)?;
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

impl ::protobuf::MessageStatic for ScanInfo {
    fn new() -> ScanInfo {
        ScanInfo::new()
    }

    fn descriptor_static(_: ::std::option::Option<ScanInfo>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeInt64>(
                    "total",
                    ScanInfo::get_total_for_reflect,
                    ScanInfo::mut_total_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeInt64>(
                    "processed",
                    ScanInfo::get_processed_for_reflect,
                    ScanInfo::mut_processed_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ScanInfo>(
                    "ScanInfo",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ScanInfo {
    fn clear(&mut self) {
        self.clear_total();
        self.clear_processed();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ScanInfo {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ScanInfo {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ScanDetail {
    // message fields
    pub write: ::protobuf::SingularPtrField<ScanInfo>,
    pub lock: ::protobuf::SingularPtrField<ScanInfo>,
    pub data: ::protobuf::SingularPtrField<ScanInfo>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ScanDetail {}

impl ScanDetail {
    pub fn new() -> ScanDetail {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ScanDetail {
        static mut instance: ::protobuf::lazy::Lazy<ScanDetail> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ScanDetail,
        };
        unsafe {
            instance.get(ScanDetail::new)
        }
    }

    // .kvrpcpb.ScanInfo write = 1;

    pub fn clear_write(&mut self) {
        self.write.clear();
    }

    pub fn has_write(&self) -> bool {
        self.write.is_some()
    }

    // Param is passed by value, moved
    pub fn set_write(&mut self, v: ScanInfo) {
        self.write = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_write(&mut self) -> &mut ScanInfo {
        if self.write.is_none() {
            self.write.set_default();
        }
        self.write.as_mut().unwrap()
    }

    // Take field
    pub fn take_write(&mut self) -> ScanInfo {
        self.write.take().unwrap_or_else(|| ScanInfo::new())
    }

    pub fn get_write(&self) -> &ScanInfo {
        self.write.as_ref().unwrap_or_else(|| ScanInfo::default_instance())
    }

    fn get_write_for_reflect(&self) -> &::protobuf::SingularPtrField<ScanInfo> {
        &self.write
    }

    fn mut_write_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<ScanInfo> {
        &mut self.write
    }

    // .kvrpcpb.ScanInfo lock = 2;

    pub fn clear_lock(&mut self) {
        self.lock.clear();
    }

    pub fn has_lock(&self) -> bool {
        self.lock.is_some()
    }

    // Param is passed by value, moved
    pub fn set_lock(&mut self, v: ScanInfo) {
        self.lock = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_lock(&mut self) -> &mut ScanInfo {
        if self.lock.is_none() {
            self.lock.set_default();
        }
        self.lock.as_mut().unwrap()
    }

    // Take field
    pub fn take_lock(&mut self) -> ScanInfo {
        self.lock.take().unwrap_or_else(|| ScanInfo::new())
    }

    pub fn get_lock(&self) -> &ScanInfo {
        self.lock.as_ref().unwrap_or_else(|| ScanInfo::default_instance())
    }

    fn get_lock_for_reflect(&self) -> &::protobuf::SingularPtrField<ScanInfo> {
        &self.lock
    }

    fn mut_lock_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<ScanInfo> {
        &mut self.lock
    }

    // .kvrpcpb.ScanInfo data = 3;

    pub fn clear_data(&mut self) {
        self.data.clear();
    }

    pub fn has_data(&self) -> bool {
        self.data.is_some()
    }

    // Param is passed by value, moved
    pub fn set_data(&mut self, v: ScanInfo) {
        self.data = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_data(&mut self) -> &mut ScanInfo {
        if self.data.is_none() {
            self.data.set_default();
        }
        self.data.as_mut().unwrap()
    }

    // Take field
    pub fn take_data(&mut self) -> ScanInfo {
        self.data.take().unwrap_or_else(|| ScanInfo::new())
    }

    pub fn get_data(&self) -> &ScanInfo {
        self.data.as_ref().unwrap_or_else(|| ScanInfo::default_instance())
    }

    fn get_data_for_reflect(&self) -> &::protobuf::SingularPtrField<ScanInfo> {
        &self.data
    }

    fn mut_data_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<ScanInfo> {
        &mut self.data
    }
}

impl ::protobuf::Message for ScanDetail {
    fn is_initialized(&self) -> bool {
        for v in &self.write {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.lock {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.data {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.write)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.lock)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.data)?;
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
        if let Some(ref v) = self.write.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.lock.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.data.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.write.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.lock.as_ref() {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.data.as_ref() {
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

impl ::protobuf::MessageStatic for ScanDetail {
    fn new() -> ScanDetail {
        ScanDetail::new()
    }

    fn descriptor_static(_: ::std::option::Option<ScanDetail>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<ScanInfo>>(
                    "write",
                    ScanDetail::get_write_for_reflect,
                    ScanDetail::mut_write_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<ScanInfo>>(
                    "lock",
                    ScanDetail::get_lock_for_reflect,
                    ScanDetail::mut_lock_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<ScanInfo>>(
                    "data",
                    ScanDetail::get_data_for_reflect,
                    ScanDetail::mut_data_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ScanDetail>(
                    "ScanDetail",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ScanDetail {
    fn clear(&mut self) {
        self.clear_write();
        self.clear_lock();
        self.clear_data();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ScanDetail {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ScanDetail {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ExecDetails {
    // message fields
    pub handle_time: ::protobuf::SingularPtrField<HandleTime>,
    pub scan_detail: ::protobuf::SingularPtrField<ScanDetail>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ExecDetails {}

impl ExecDetails {
    pub fn new() -> ExecDetails {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ExecDetails {
        static mut instance: ::protobuf::lazy::Lazy<ExecDetails> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ExecDetails,
        };
        unsafe {
            instance.get(ExecDetails::new)
        }
    }

    // .kvrpcpb.HandleTime handle_time = 1;

    pub fn clear_handle_time(&mut self) {
        self.handle_time.clear();
    }

    pub fn has_handle_time(&self) -> bool {
        self.handle_time.is_some()
    }

    // Param is passed by value, moved
    pub fn set_handle_time(&mut self, v: HandleTime) {
        self.handle_time = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_handle_time(&mut self) -> &mut HandleTime {
        if self.handle_time.is_none() {
            self.handle_time.set_default();
        }
        self.handle_time.as_mut().unwrap()
    }

    // Take field
    pub fn take_handle_time(&mut self) -> HandleTime {
        self.handle_time.take().unwrap_or_else(|| HandleTime::new())
    }

    pub fn get_handle_time(&self) -> &HandleTime {
        self.handle_time.as_ref().unwrap_or_else(|| HandleTime::default_instance())
    }

    fn get_handle_time_for_reflect(&self) -> &::protobuf::SingularPtrField<HandleTime> {
        &self.handle_time
    }

    fn mut_handle_time_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<HandleTime> {
        &mut self.handle_time
    }

    // .kvrpcpb.ScanDetail scan_detail = 2;

    pub fn clear_scan_detail(&mut self) {
        self.scan_detail.clear();
    }

    pub fn has_scan_detail(&self) -> bool {
        self.scan_detail.is_some()
    }

    // Param is passed by value, moved
    pub fn set_scan_detail(&mut self, v: ScanDetail) {
        self.scan_detail = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_scan_detail(&mut self) -> &mut ScanDetail {
        if self.scan_detail.is_none() {
            self.scan_detail.set_default();
        }
        self.scan_detail.as_mut().unwrap()
    }

    // Take field
    pub fn take_scan_detail(&mut self) -> ScanDetail {
        self.scan_detail.take().unwrap_or_else(|| ScanDetail::new())
    }

    pub fn get_scan_detail(&self) -> &ScanDetail {
        self.scan_detail.as_ref().unwrap_or_else(|| ScanDetail::default_instance())
    }

    fn get_scan_detail_for_reflect(&self) -> &::protobuf::SingularPtrField<ScanDetail> {
        &self.scan_detail
    }

    fn mut_scan_detail_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<ScanDetail> {
        &mut self.scan_detail
    }
}

impl ::protobuf::Message for ExecDetails {
    fn is_initialized(&self) -> bool {
        for v in &self.handle_time {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.scan_detail {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.handle_time)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.scan_detail)?;
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
        if let Some(ref v) = self.handle_time.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.scan_detail.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.handle_time.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.scan_detail.as_ref() {
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

impl ::protobuf::MessageStatic for ExecDetails {
    fn new() -> ExecDetails {
        ExecDetails::new()
    }

    fn descriptor_static(_: ::std::option::Option<ExecDetails>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<HandleTime>>(
                    "handle_time",
                    ExecDetails::get_handle_time_for_reflect,
                    ExecDetails::mut_handle_time_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<ScanDetail>>(
                    "scan_detail",
                    ExecDetails::get_scan_detail_for_reflect,
                    ExecDetails::mut_scan_detail_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ExecDetails>(
                    "ExecDetails",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ExecDetails {
    fn clear(&mut self) {
        self.clear_handle_time();
        self.clear_scan_detail();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ExecDetails {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ExecDetails {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct GetRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub key: ::std::vec::Vec<u8>,
    pub version: u64,
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

    // .kvrpcpb.Context context = 1;

    pub fn clear_context(&mut self) {
        self.context.clear();
    }

    pub fn has_context(&self) -> bool {
        self.context.is_some()
    }

    // Param is passed by value, moved
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
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

    // uint64 version = 3;

    pub fn clear_version(&mut self) {
        self.version = 0;
    }

    // Param is passed by value, moved
    pub fn set_version(&mut self, v: u64) {
        self.version = v;
    }

    pub fn get_version(&self) -> u64 {
        self.version
    }

    fn get_version_for_reflect(&self) -> &u64 {
        &self.version
    }

    fn mut_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.version
    }
}

impl ::protobuf::Message for GetRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.version = tmp;
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
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.key);
        }
        if self.version != 0 {
            my_size += ::protobuf::rt::value_size(3, self.version, ::protobuf::wire_format::WireTypeVarint);
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
        if !self.key.is_empty() {
            os.write_bytes(2, &self.key)?;
        }
        if self.version != 0 {
            os.write_uint64(3, self.version)?;
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
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    GetRequest::get_context_for_reflect,
                    GetRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    GetRequest::get_key_for_reflect,
                    GetRequest::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "version",
                    GetRequest::get_version_for_reflect,
                    GetRequest::mut_version_for_reflect,
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
        self.clear_context();
        self.clear_key();
        self.clear_version();
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
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::protobuf::SingularPtrField<KeyError>,
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

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // .kvrpcpb.KeyError error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: KeyError) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut KeyError {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> KeyError {
        self.error.take().unwrap_or_else(|| KeyError::new())
    }

    pub fn get_error(&self) -> &KeyError {
        self.error.as_ref().unwrap_or_else(|| KeyError::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<KeyError> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<KeyError> {
        &mut self.error
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

impl ::protobuf::Message for GetResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.error)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.value.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.value);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.error.as_ref() {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
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
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    GetResponse::get_region_error_for_reflect,
                    GetResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyError>>(
                    "error",
                    GetResponse::get_error_for_reflect,
                    GetResponse::mut_error_for_reflect,
                ));
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
        self.clear_region_error();
        self.clear_error();
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
pub struct ScanRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub start_key: ::std::vec::Vec<u8>,
    pub limit: u32,
    pub version: u64,
    pub key_only: bool,
    pub reverse: bool,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ScanRequest {}

impl ScanRequest {
    pub fn new() -> ScanRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ScanRequest {
        static mut instance: ::protobuf::lazy::Lazy<ScanRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ScanRequest,
        };
        unsafe {
            instance.get(ScanRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // bytes start_key = 2;

    pub fn clear_start_key(&mut self) {
        self.start_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_start_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.start_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_start_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // Take field
    pub fn take_start_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.start_key, ::std::vec::Vec::new())
    }

    pub fn get_start_key(&self) -> &[u8] {
        &self.start_key
    }

    fn get_start_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.start_key
    }

    fn mut_start_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // uint32 limit = 3;

    pub fn clear_limit(&mut self) {
        self.limit = 0;
    }

    // Param is passed by value, moved
    pub fn set_limit(&mut self, v: u32) {
        self.limit = v;
    }

    pub fn get_limit(&self) -> u32 {
        self.limit
    }

    fn get_limit_for_reflect(&self) -> &u32 {
        &self.limit
    }

    fn mut_limit_for_reflect(&mut self) -> &mut u32 {
        &mut self.limit
    }

    // uint64 version = 4;

    pub fn clear_version(&mut self) {
        self.version = 0;
    }

    // Param is passed by value, moved
    pub fn set_version(&mut self, v: u64) {
        self.version = v;
    }

    pub fn get_version(&self) -> u64 {
        self.version
    }

    fn get_version_for_reflect(&self) -> &u64 {
        &self.version
    }

    fn mut_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.version
    }

    // bool key_only = 5;

    pub fn clear_key_only(&mut self) {
        self.key_only = false;
    }

    // Param is passed by value, moved
    pub fn set_key_only(&mut self, v: bool) {
        self.key_only = v;
    }

    pub fn get_key_only(&self) -> bool {
        self.key_only
    }

    fn get_key_only_for_reflect(&self) -> &bool {
        &self.key_only
    }

    fn mut_key_only_for_reflect(&mut self) -> &mut bool {
        &mut self.key_only
    }

    // bool reverse = 6;

    pub fn clear_reverse(&mut self) {
        self.reverse = false;
    }

    // Param is passed by value, moved
    pub fn set_reverse(&mut self, v: bool) {
        self.reverse = v;
    }

    pub fn get_reverse(&self) -> bool {
        self.reverse
    }

    fn get_reverse_for_reflect(&self) -> &bool {
        &self.reverse
    }

    fn mut_reverse_for_reflect(&mut self) -> &mut bool {
        &mut self.reverse
    }
}

impl ::protobuf::Message for ScanRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.start_key)?;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint32()?;
                    self.limit = tmp;
                },
                4 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.version = tmp;
                },
                5 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_bool()?;
                    self.key_only = tmp;
                },
                6 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_bool()?;
                    self.reverse = tmp;
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
        if !self.start_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.start_key);
        }
        if self.limit != 0 {
            my_size += ::protobuf::rt::value_size(3, self.limit, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.version != 0 {
            my_size += ::protobuf::rt::value_size(4, self.version, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.key_only != false {
            my_size += 2;
        }
        if self.reverse != false {
            my_size += 2;
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
        if !self.start_key.is_empty() {
            os.write_bytes(2, &self.start_key)?;
        }
        if self.limit != 0 {
            os.write_uint32(3, self.limit)?;
        }
        if self.version != 0 {
            os.write_uint64(4, self.version)?;
        }
        if self.key_only != false {
            os.write_bool(5, self.key_only)?;
        }
        if self.reverse != false {
            os.write_bool(6, self.reverse)?;
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

impl ::protobuf::MessageStatic for ScanRequest {
    fn new() -> ScanRequest {
        ScanRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<ScanRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    ScanRequest::get_context_for_reflect,
                    ScanRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "start_key",
                    ScanRequest::get_start_key_for_reflect,
                    ScanRequest::mut_start_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint32>(
                    "limit",
                    ScanRequest::get_limit_for_reflect,
                    ScanRequest::mut_limit_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "version",
                    ScanRequest::get_version_for_reflect,
                    ScanRequest::mut_version_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "key_only",
                    ScanRequest::get_key_only_for_reflect,
                    ScanRequest::mut_key_only_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "reverse",
                    ScanRequest::get_reverse_for_reflect,
                    ScanRequest::mut_reverse_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ScanRequest>(
                    "ScanRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ScanRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_start_key();
        self.clear_limit();
        self.clear_version();
        self.clear_key_only();
        self.clear_reverse();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ScanRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ScanRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct KvPair {
    // message fields
    pub error: ::protobuf::SingularPtrField<KeyError>,
    pub key: ::std::vec::Vec<u8>,
    pub value: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for KvPair {}

impl KvPair {
    pub fn new() -> KvPair {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static KvPair {
        static mut instance: ::protobuf::lazy::Lazy<KvPair> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const KvPair,
        };
        unsafe {
            instance.get(KvPair::new)
        }
    }

    // .kvrpcpb.KeyError error = 1;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: KeyError) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut KeyError {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> KeyError {
        self.error.take().unwrap_or_else(|| KeyError::new())
    }

    pub fn get_error(&self) -> &KeyError {
        self.error.as_ref().unwrap_or_else(|| KeyError::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<KeyError> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<KeyError> {
        &mut self.error
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

impl ::protobuf::Message for KvPair {
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
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
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
        if let Some(ref v) = self.error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
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

impl ::protobuf::MessageStatic for KvPair {
    fn new() -> KvPair {
        KvPair::new()
    }

    fn descriptor_static(_: ::std::option::Option<KvPair>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyError>>(
                    "error",
                    KvPair::get_error_for_reflect,
                    KvPair::mut_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    KvPair::get_key_for_reflect,
                    KvPair::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "value",
                    KvPair::get_value_for_reflect,
                    KvPair::mut_value_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<KvPair>(
                    "KvPair",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for KvPair {
    fn clear(&mut self) {
        self.clear_error();
        self.clear_key();
        self.clear_value();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for KvPair {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for KvPair {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ScanResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub pairs: ::protobuf::RepeatedField<KvPair>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ScanResponse {}

impl ScanResponse {
    pub fn new() -> ScanResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ScanResponse {
        static mut instance: ::protobuf::lazy::Lazy<ScanResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ScanResponse,
        };
        unsafe {
            instance.get(ScanResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // repeated .kvrpcpb.KvPair pairs = 2;

    pub fn clear_pairs(&mut self) {
        self.pairs.clear();
    }

    // Param is passed by value, moved
    pub fn set_pairs(&mut self, v: ::protobuf::RepeatedField<KvPair>) {
        self.pairs = v;
    }

    // Mutable pointer to the field.
    pub fn mut_pairs(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.pairs
    }

    // Take field
    pub fn take_pairs(&mut self) -> ::protobuf::RepeatedField<KvPair> {
        ::std::mem::replace(&mut self.pairs, ::protobuf::RepeatedField::new())
    }

    pub fn get_pairs(&self) -> &[KvPair] {
        &self.pairs
    }

    fn get_pairs_for_reflect(&self) -> &::protobuf::RepeatedField<KvPair> {
        &self.pairs
    }

    fn mut_pairs_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.pairs
    }
}

impl ::protobuf::Message for ScanResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.pairs {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.pairs)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        for value in &self.pairs {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        for v in &self.pairs {
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

impl ::protobuf::MessageStatic for ScanResponse {
    fn new() -> ScanResponse {
        ScanResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<ScanResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    ScanResponse::get_region_error_for_reflect,
                    ScanResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KvPair>>(
                    "pairs",
                    ScanResponse::get_pairs_for_reflect,
                    ScanResponse::mut_pairs_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ScanResponse>(
                    "ScanResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ScanResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_pairs();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ScanResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ScanResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct Mutation {
    // message fields
    pub op: Op,
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

    // .kvrpcpb.Op op = 1;

    pub fn clear_op(&mut self) {
        self.op = Op::Put;
    }

    // Param is passed by value, moved
    pub fn set_op(&mut self, v: Op) {
        self.op = v;
    }

    pub fn get_op(&self) -> Op {
        self.op
    }

    fn get_op_for_reflect(&self) -> &Op {
        &self.op
    }

    fn mut_op_for_reflect(&mut self) -> &mut Op {
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
        if self.op != Op::Put {
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
        if self.op != Op::Put {
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
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeEnum<Op>>(
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

#[derive(PartialEq,Clone,Default)]
pub struct PrewriteRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub mutations: ::protobuf::RepeatedField<Mutation>,
    pub primary_lock: ::std::vec::Vec<u8>,
    pub start_version: u64,
    pub lock_ttl: u64,
    pub skip_constraint_check: bool,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for PrewriteRequest {}

impl PrewriteRequest {
    pub fn new() -> PrewriteRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static PrewriteRequest {
        static mut instance: ::protobuf::lazy::Lazy<PrewriteRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const PrewriteRequest,
        };
        unsafe {
            instance.get(PrewriteRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // repeated .kvrpcpb.Mutation mutations = 2;

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

    // bytes primary_lock = 3;

    pub fn clear_primary_lock(&mut self) {
        self.primary_lock.clear();
    }

    // Param is passed by value, moved
    pub fn set_primary_lock(&mut self, v: ::std::vec::Vec<u8>) {
        self.primary_lock = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_primary_lock(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.primary_lock
    }

    // Take field
    pub fn take_primary_lock(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.primary_lock, ::std::vec::Vec::new())
    }

    pub fn get_primary_lock(&self) -> &[u8] {
        &self.primary_lock
    }

    fn get_primary_lock_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.primary_lock
    }

    fn mut_primary_lock_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.primary_lock
    }

    // uint64 start_version = 4;

    pub fn clear_start_version(&mut self) {
        self.start_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_version(&mut self, v: u64) {
        self.start_version = v;
    }

    pub fn get_start_version(&self) -> u64 {
        self.start_version
    }

    fn get_start_version_for_reflect(&self) -> &u64 {
        &self.start_version
    }

    fn mut_start_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_version
    }

    // uint64 lock_ttl = 5;

    pub fn clear_lock_ttl(&mut self) {
        self.lock_ttl = 0;
    }

    // Param is passed by value, moved
    pub fn set_lock_ttl(&mut self, v: u64) {
        self.lock_ttl = v;
    }

    pub fn get_lock_ttl(&self) -> u64 {
        self.lock_ttl
    }

    fn get_lock_ttl_for_reflect(&self) -> &u64 {
        &self.lock_ttl
    }

    fn mut_lock_ttl_for_reflect(&mut self) -> &mut u64 {
        &mut self.lock_ttl
    }

    // bool skip_constraint_check = 6;

    pub fn clear_skip_constraint_check(&mut self) {
        self.skip_constraint_check = false;
    }

    // Param is passed by value, moved
    pub fn set_skip_constraint_check(&mut self, v: bool) {
        self.skip_constraint_check = v;
    }

    pub fn get_skip_constraint_check(&self) -> bool {
        self.skip_constraint_check
    }

    fn get_skip_constraint_check_for_reflect(&self) -> &bool {
        &self.skip_constraint_check
    }

    fn mut_skip_constraint_check_for_reflect(&mut self) -> &mut bool {
        &mut self.skip_constraint_check
    }
}

impl ::protobuf::Message for PrewriteRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
            if !v.is_initialized() {
                return false;
            }
        };
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.context)?;
                },
                2 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.mutations)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.primary_lock)?;
                },
                4 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.start_version = tmp;
                },
                5 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.lock_ttl = tmp;
                },
                6 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_bool()?;
                    self.skip_constraint_check = tmp;
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
        for value in &self.mutations {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        if !self.primary_lock.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.primary_lock);
        }
        if self.start_version != 0 {
            my_size += ::protobuf::rt::value_size(4, self.start_version, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.lock_ttl != 0 {
            my_size += ::protobuf::rt::value_size(5, self.lock_ttl, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.skip_constraint_check != false {
            my_size += 2;
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
        for v in &self.mutations {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        };
        if !self.primary_lock.is_empty() {
            os.write_bytes(3, &self.primary_lock)?;
        }
        if self.start_version != 0 {
            os.write_uint64(4, self.start_version)?;
        }
        if self.lock_ttl != 0 {
            os.write_uint64(5, self.lock_ttl)?;
        }
        if self.skip_constraint_check != false {
            os.write_bool(6, self.skip_constraint_check)?;
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

impl ::protobuf::MessageStatic for PrewriteRequest {
    fn new() -> PrewriteRequest {
        PrewriteRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<PrewriteRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    PrewriteRequest::get_context_for_reflect,
                    PrewriteRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Mutation>>(
                    "mutations",
                    PrewriteRequest::get_mutations_for_reflect,
                    PrewriteRequest::mut_mutations_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "primary_lock",
                    PrewriteRequest::get_primary_lock_for_reflect,
                    PrewriteRequest::mut_primary_lock_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_version",
                    PrewriteRequest::get_start_version_for_reflect,
                    PrewriteRequest::mut_start_version_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "lock_ttl",
                    PrewriteRequest::get_lock_ttl_for_reflect,
                    PrewriteRequest::mut_lock_ttl_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "skip_constraint_check",
                    PrewriteRequest::get_skip_constraint_check_for_reflect,
                    PrewriteRequest::mut_skip_constraint_check_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<PrewriteRequest>(
                    "PrewriteRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for PrewriteRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_mutations();
        self.clear_primary_lock();
        self.clear_start_version();
        self.clear_lock_ttl();
        self.clear_skip_constraint_check();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for PrewriteRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for PrewriteRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct PrewriteResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub errors: ::protobuf::RepeatedField<KeyError>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for PrewriteResponse {}

impl PrewriteResponse {
    pub fn new() -> PrewriteResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static PrewriteResponse {
        static mut instance: ::protobuf::lazy::Lazy<PrewriteResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const PrewriteResponse,
        };
        unsafe {
            instance.get(PrewriteResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // repeated .kvrpcpb.KeyError errors = 2;

    pub fn clear_errors(&mut self) {
        self.errors.clear();
    }

    // Param is passed by value, moved
    pub fn set_errors(&mut self, v: ::protobuf::RepeatedField<KeyError>) {
        self.errors = v;
    }

    // Mutable pointer to the field.
    pub fn mut_errors(&mut self) -> &mut ::protobuf::RepeatedField<KeyError> {
        &mut self.errors
    }

    // Take field
    pub fn take_errors(&mut self) -> ::protobuf::RepeatedField<KeyError> {
        ::std::mem::replace(&mut self.errors, ::protobuf::RepeatedField::new())
    }

    pub fn get_errors(&self) -> &[KeyError] {
        &self.errors
    }

    fn get_errors_for_reflect(&self) -> &::protobuf::RepeatedField<KeyError> {
        &self.errors
    }

    fn mut_errors_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<KeyError> {
        &mut self.errors
    }
}

impl ::protobuf::Message for PrewriteResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.errors {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.errors)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        for value in &self.errors {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        for v in &self.errors {
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

impl ::protobuf::MessageStatic for PrewriteResponse {
    fn new() -> PrewriteResponse {
        PrewriteResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<PrewriteResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    PrewriteResponse::get_region_error_for_reflect,
                    PrewriteResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyError>>(
                    "errors",
                    PrewriteResponse::get_errors_for_reflect,
                    PrewriteResponse::mut_errors_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<PrewriteResponse>(
                    "PrewriteResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for PrewriteResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_errors();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for PrewriteResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for PrewriteResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct CommitRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub start_version: u64,
    pub keys: ::protobuf::RepeatedField<::std::vec::Vec<u8>>,
    pub commit_version: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for CommitRequest {}

impl CommitRequest {
    pub fn new() -> CommitRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static CommitRequest {
        static mut instance: ::protobuf::lazy::Lazy<CommitRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const CommitRequest,
        };
        unsafe {
            instance.get(CommitRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // uint64 start_version = 2;

    pub fn clear_start_version(&mut self) {
        self.start_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_version(&mut self, v: u64) {
        self.start_version = v;
    }

    pub fn get_start_version(&self) -> u64 {
        self.start_version
    }

    fn get_start_version_for_reflect(&self) -> &u64 {
        &self.start_version
    }

    fn mut_start_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_version
    }

    // repeated bytes keys = 3;

    pub fn clear_keys(&mut self) {
        self.keys.clear();
    }

    // Param is passed by value, moved
    pub fn set_keys(&mut self, v: ::protobuf::RepeatedField<::std::vec::Vec<u8>>) {
        self.keys = v;
    }

    // Mutable pointer to the field.
    pub fn mut_keys(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }

    // Take field
    pub fn take_keys(&mut self) -> ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        ::std::mem::replace(&mut self.keys, ::protobuf::RepeatedField::new())
    }

    pub fn get_keys(&self) -> &[::std::vec::Vec<u8>] {
        &self.keys
    }

    fn get_keys_for_reflect(&self) -> &::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &self.keys
    }

    fn mut_keys_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }

    // uint64 commit_version = 4;

    pub fn clear_commit_version(&mut self) {
        self.commit_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_commit_version(&mut self, v: u64) {
        self.commit_version = v;
    }

    pub fn get_commit_version(&self) -> u64 {
        self.commit_version
    }

    fn get_commit_version_for_reflect(&self) -> &u64 {
        &self.commit_version
    }

    fn mut_commit_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.commit_version
    }
}

impl ::protobuf::Message for CommitRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.start_version = tmp;
                },
                3 => {
                    ::protobuf::rt::read_repeated_bytes_into(wire_type, is, &mut self.keys)?;
                },
                4 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.commit_version = tmp;
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
        if self.start_version != 0 {
            my_size += ::protobuf::rt::value_size(2, self.start_version, ::protobuf::wire_format::WireTypeVarint);
        }
        for value in &self.keys {
            my_size += ::protobuf::rt::bytes_size(3, &value);
        };
        if self.commit_version != 0 {
            my_size += ::protobuf::rt::value_size(4, self.commit_version, ::protobuf::wire_format::WireTypeVarint);
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
        if self.start_version != 0 {
            os.write_uint64(2, self.start_version)?;
        }
        for v in &self.keys {
            os.write_bytes(3, &v)?;
        };
        if self.commit_version != 0 {
            os.write_uint64(4, self.commit_version)?;
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

impl ::protobuf::MessageStatic for CommitRequest {
    fn new() -> CommitRequest {
        CommitRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<CommitRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    CommitRequest::get_context_for_reflect,
                    CommitRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_version",
                    CommitRequest::get_start_version_for_reflect,
                    CommitRequest::mut_start_version_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "keys",
                    CommitRequest::get_keys_for_reflect,
                    CommitRequest::mut_keys_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "commit_version",
                    CommitRequest::get_commit_version_for_reflect,
                    CommitRequest::mut_commit_version_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<CommitRequest>(
                    "CommitRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for CommitRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_start_version();
        self.clear_keys();
        self.clear_commit_version();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for CommitRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for CommitRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct CommitResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::protobuf::SingularPtrField<KeyError>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for CommitResponse {}

impl CommitResponse {
    pub fn new() -> CommitResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static CommitResponse {
        static mut instance: ::protobuf::lazy::Lazy<CommitResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const CommitResponse,
        };
        unsafe {
            instance.get(CommitResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // .kvrpcpb.KeyError error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: KeyError) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut KeyError {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> KeyError {
        self.error.take().unwrap_or_else(|| KeyError::new())
    }

    pub fn get_error(&self) -> &KeyError {
        self.error.as_ref().unwrap_or_else(|| KeyError::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<KeyError> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<KeyError> {
        &mut self.error
    }
}

impl ::protobuf::Message for CommitResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.error.as_ref() {
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

impl ::protobuf::MessageStatic for CommitResponse {
    fn new() -> CommitResponse {
        CommitResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<CommitResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    CommitResponse::get_region_error_for_reflect,
                    CommitResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyError>>(
                    "error",
                    CommitResponse::get_error_for_reflect,
                    CommitResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<CommitResponse>(
                    "CommitResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for CommitResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for CommitResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for CommitResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ImportRequest {
    // message fields
    pub mutations: ::protobuf::RepeatedField<Mutation>,
    pub commit_version: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ImportRequest {}

impl ImportRequest {
    pub fn new() -> ImportRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ImportRequest {
        static mut instance: ::protobuf::lazy::Lazy<ImportRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ImportRequest,
        };
        unsafe {
            instance.get(ImportRequest::new)
        }
    }

    // repeated .kvrpcpb.Mutation mutations = 1;

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

    // uint64 commit_version = 2;

    pub fn clear_commit_version(&mut self) {
        self.commit_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_commit_version(&mut self, v: u64) {
        self.commit_version = v;
    }

    pub fn get_commit_version(&self) -> u64 {
        self.commit_version
    }

    fn get_commit_version_for_reflect(&self) -> &u64 {
        &self.commit_version
    }

    fn mut_commit_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.commit_version
    }
}

impl ::protobuf::Message for ImportRequest {
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
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.mutations)?;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.commit_version = tmp;
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
        for value in &self.mutations {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        if self.commit_version != 0 {
            my_size += ::protobuf::rt::value_size(2, self.commit_version, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        for v in &self.mutations {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        };
        if self.commit_version != 0 {
            os.write_uint64(2, self.commit_version)?;
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

impl ::protobuf::MessageStatic for ImportRequest {
    fn new() -> ImportRequest {
        ImportRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<ImportRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Mutation>>(
                    "mutations",
                    ImportRequest::get_mutations_for_reflect,
                    ImportRequest::mut_mutations_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "commit_version",
                    ImportRequest::get_commit_version_for_reflect,
                    ImportRequest::mut_commit_version_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ImportRequest>(
                    "ImportRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ImportRequest {
    fn clear(&mut self) {
        self.clear_mutations();
        self.clear_commit_version();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ImportRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ImportRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ImportResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ImportResponse {}

impl ImportResponse {
    pub fn new() -> ImportResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ImportResponse {
        static mut instance: ::protobuf::lazy::Lazy<ImportResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ImportResponse,
        };
        unsafe {
            instance.get(ImportResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }
}

impl ::protobuf::Message for ImportResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
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

impl ::protobuf::MessageStatic for ImportResponse {
    fn new() -> ImportResponse {
        ImportResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<ImportResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    ImportResponse::get_region_error_for_reflect,
                    ImportResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    ImportResponse::get_error_for_reflect,
                    ImportResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ImportResponse>(
                    "ImportResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ImportResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ImportResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ImportResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct BatchRollbackRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub start_version: u64,
    pub keys: ::protobuf::RepeatedField<::std::vec::Vec<u8>>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for BatchRollbackRequest {}

impl BatchRollbackRequest {
    pub fn new() -> BatchRollbackRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static BatchRollbackRequest {
        static mut instance: ::protobuf::lazy::Lazy<BatchRollbackRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const BatchRollbackRequest,
        };
        unsafe {
            instance.get(BatchRollbackRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // uint64 start_version = 2;

    pub fn clear_start_version(&mut self) {
        self.start_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_version(&mut self, v: u64) {
        self.start_version = v;
    }

    pub fn get_start_version(&self) -> u64 {
        self.start_version
    }

    fn get_start_version_for_reflect(&self) -> &u64 {
        &self.start_version
    }

    fn mut_start_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_version
    }

    // repeated bytes keys = 3;

    pub fn clear_keys(&mut self) {
        self.keys.clear();
    }

    // Param is passed by value, moved
    pub fn set_keys(&mut self, v: ::protobuf::RepeatedField<::std::vec::Vec<u8>>) {
        self.keys = v;
    }

    // Mutable pointer to the field.
    pub fn mut_keys(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }

    // Take field
    pub fn take_keys(&mut self) -> ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        ::std::mem::replace(&mut self.keys, ::protobuf::RepeatedField::new())
    }

    pub fn get_keys(&self) -> &[::std::vec::Vec<u8>] {
        &self.keys
    }

    fn get_keys_for_reflect(&self) -> &::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &self.keys
    }

    fn mut_keys_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }
}

impl ::protobuf::Message for BatchRollbackRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.start_version = tmp;
                },
                3 => {
                    ::protobuf::rt::read_repeated_bytes_into(wire_type, is, &mut self.keys)?;
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
        if self.start_version != 0 {
            my_size += ::protobuf::rt::value_size(2, self.start_version, ::protobuf::wire_format::WireTypeVarint);
        }
        for value in &self.keys {
            my_size += ::protobuf::rt::bytes_size(3, &value);
        };
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
        if self.start_version != 0 {
            os.write_uint64(2, self.start_version)?;
        }
        for v in &self.keys {
            os.write_bytes(3, &v)?;
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

impl ::protobuf::MessageStatic for BatchRollbackRequest {
    fn new() -> BatchRollbackRequest {
        BatchRollbackRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<BatchRollbackRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    BatchRollbackRequest::get_context_for_reflect,
                    BatchRollbackRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_version",
                    BatchRollbackRequest::get_start_version_for_reflect,
                    BatchRollbackRequest::mut_start_version_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "keys",
                    BatchRollbackRequest::get_keys_for_reflect,
                    BatchRollbackRequest::mut_keys_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<BatchRollbackRequest>(
                    "BatchRollbackRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for BatchRollbackRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_start_version();
        self.clear_keys();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for BatchRollbackRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for BatchRollbackRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct BatchRollbackResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::protobuf::SingularPtrField<KeyError>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for BatchRollbackResponse {}

impl BatchRollbackResponse {
    pub fn new() -> BatchRollbackResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static BatchRollbackResponse {
        static mut instance: ::protobuf::lazy::Lazy<BatchRollbackResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const BatchRollbackResponse,
        };
        unsafe {
            instance.get(BatchRollbackResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // .kvrpcpb.KeyError error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: KeyError) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut KeyError {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> KeyError {
        self.error.take().unwrap_or_else(|| KeyError::new())
    }

    pub fn get_error(&self) -> &KeyError {
        self.error.as_ref().unwrap_or_else(|| KeyError::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<KeyError> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<KeyError> {
        &mut self.error
    }
}

impl ::protobuf::Message for BatchRollbackResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.error.as_ref() {
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

impl ::protobuf::MessageStatic for BatchRollbackResponse {
    fn new() -> BatchRollbackResponse {
        BatchRollbackResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<BatchRollbackResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    BatchRollbackResponse::get_region_error_for_reflect,
                    BatchRollbackResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyError>>(
                    "error",
                    BatchRollbackResponse::get_error_for_reflect,
                    BatchRollbackResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<BatchRollbackResponse>(
                    "BatchRollbackResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for BatchRollbackResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for BatchRollbackResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for BatchRollbackResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct CleanupRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub key: ::std::vec::Vec<u8>,
    pub start_version: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for CleanupRequest {}

impl CleanupRequest {
    pub fn new() -> CleanupRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static CleanupRequest {
        static mut instance: ::protobuf::lazy::Lazy<CleanupRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const CleanupRequest,
        };
        unsafe {
            instance.get(CleanupRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
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

    // uint64 start_version = 3;

    pub fn clear_start_version(&mut self) {
        self.start_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_version(&mut self, v: u64) {
        self.start_version = v;
    }

    pub fn get_start_version(&self) -> u64 {
        self.start_version
    }

    fn get_start_version_for_reflect(&self) -> &u64 {
        &self.start_version
    }

    fn mut_start_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_version
    }
}

impl ::protobuf::Message for CleanupRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.start_version = tmp;
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
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.key);
        }
        if self.start_version != 0 {
            my_size += ::protobuf::rt::value_size(3, self.start_version, ::protobuf::wire_format::WireTypeVarint);
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
        if !self.key.is_empty() {
            os.write_bytes(2, &self.key)?;
        }
        if self.start_version != 0 {
            os.write_uint64(3, self.start_version)?;
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

impl ::protobuf::MessageStatic for CleanupRequest {
    fn new() -> CleanupRequest {
        CleanupRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<CleanupRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    CleanupRequest::get_context_for_reflect,
                    CleanupRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    CleanupRequest::get_key_for_reflect,
                    CleanupRequest::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_version",
                    CleanupRequest::get_start_version_for_reflect,
                    CleanupRequest::mut_start_version_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<CleanupRequest>(
                    "CleanupRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for CleanupRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_key();
        self.clear_start_version();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for CleanupRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for CleanupRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct CleanupResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::protobuf::SingularPtrField<KeyError>,
    pub commit_version: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for CleanupResponse {}

impl CleanupResponse {
    pub fn new() -> CleanupResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static CleanupResponse {
        static mut instance: ::protobuf::lazy::Lazy<CleanupResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const CleanupResponse,
        };
        unsafe {
            instance.get(CleanupResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // .kvrpcpb.KeyError error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: KeyError) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut KeyError {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> KeyError {
        self.error.take().unwrap_or_else(|| KeyError::new())
    }

    pub fn get_error(&self) -> &KeyError {
        self.error.as_ref().unwrap_or_else(|| KeyError::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<KeyError> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<KeyError> {
        &mut self.error
    }

    // uint64 commit_version = 3;

    pub fn clear_commit_version(&mut self) {
        self.commit_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_commit_version(&mut self, v: u64) {
        self.commit_version = v;
    }

    pub fn get_commit_version(&self) -> u64 {
        self.commit_version
    }

    fn get_commit_version_for_reflect(&self) -> &u64 {
        &self.commit_version
    }

    fn mut_commit_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.commit_version
    }
}

impl ::protobuf::Message for CleanupResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.error)?;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.commit_version = tmp;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if self.commit_version != 0 {
            my_size += ::protobuf::rt::value_size(3, self.commit_version, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.error.as_ref() {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if self.commit_version != 0 {
            os.write_uint64(3, self.commit_version)?;
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

impl ::protobuf::MessageStatic for CleanupResponse {
    fn new() -> CleanupResponse {
        CleanupResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<CleanupResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    CleanupResponse::get_region_error_for_reflect,
                    CleanupResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyError>>(
                    "error",
                    CleanupResponse::get_error_for_reflect,
                    CleanupResponse::mut_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "commit_version",
                    CleanupResponse::get_commit_version_for_reflect,
                    CleanupResponse::mut_commit_version_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<CleanupResponse>(
                    "CleanupResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for CleanupResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.clear_commit_version();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for CleanupResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for CleanupResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct BatchGetRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub keys: ::protobuf::RepeatedField<::std::vec::Vec<u8>>,
    pub version: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for BatchGetRequest {}

impl BatchGetRequest {
    pub fn new() -> BatchGetRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static BatchGetRequest {
        static mut instance: ::protobuf::lazy::Lazy<BatchGetRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const BatchGetRequest,
        };
        unsafe {
            instance.get(BatchGetRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // repeated bytes keys = 2;

    pub fn clear_keys(&mut self) {
        self.keys.clear();
    }

    // Param is passed by value, moved
    pub fn set_keys(&mut self, v: ::protobuf::RepeatedField<::std::vec::Vec<u8>>) {
        self.keys = v;
    }

    // Mutable pointer to the field.
    pub fn mut_keys(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }

    // Take field
    pub fn take_keys(&mut self) -> ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        ::std::mem::replace(&mut self.keys, ::protobuf::RepeatedField::new())
    }

    pub fn get_keys(&self) -> &[::std::vec::Vec<u8>] {
        &self.keys
    }

    fn get_keys_for_reflect(&self) -> &::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &self.keys
    }

    fn mut_keys_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }

    // uint64 version = 3;

    pub fn clear_version(&mut self) {
        self.version = 0;
    }

    // Param is passed by value, moved
    pub fn set_version(&mut self, v: u64) {
        self.version = v;
    }

    pub fn get_version(&self) -> u64 {
        self.version
    }

    fn get_version_for_reflect(&self) -> &u64 {
        &self.version
    }

    fn mut_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.version
    }
}

impl ::protobuf::Message for BatchGetRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_repeated_bytes_into(wire_type, is, &mut self.keys)?;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.version = tmp;
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
        for value in &self.keys {
            my_size += ::protobuf::rt::bytes_size(2, &value);
        };
        if self.version != 0 {
            my_size += ::protobuf::rt::value_size(3, self.version, ::protobuf::wire_format::WireTypeVarint);
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
        for v in &self.keys {
            os.write_bytes(2, &v)?;
        };
        if self.version != 0 {
            os.write_uint64(3, self.version)?;
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

impl ::protobuf::MessageStatic for BatchGetRequest {
    fn new() -> BatchGetRequest {
        BatchGetRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<BatchGetRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    BatchGetRequest::get_context_for_reflect,
                    BatchGetRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "keys",
                    BatchGetRequest::get_keys_for_reflect,
                    BatchGetRequest::mut_keys_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "version",
                    BatchGetRequest::get_version_for_reflect,
                    BatchGetRequest::mut_version_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<BatchGetRequest>(
                    "BatchGetRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for BatchGetRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_keys();
        self.clear_version();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for BatchGetRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for BatchGetRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct BatchGetResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub pairs: ::protobuf::RepeatedField<KvPair>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for BatchGetResponse {}

impl BatchGetResponse {
    pub fn new() -> BatchGetResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static BatchGetResponse {
        static mut instance: ::protobuf::lazy::Lazy<BatchGetResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const BatchGetResponse,
        };
        unsafe {
            instance.get(BatchGetResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // repeated .kvrpcpb.KvPair pairs = 2;

    pub fn clear_pairs(&mut self) {
        self.pairs.clear();
    }

    // Param is passed by value, moved
    pub fn set_pairs(&mut self, v: ::protobuf::RepeatedField<KvPair>) {
        self.pairs = v;
    }

    // Mutable pointer to the field.
    pub fn mut_pairs(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.pairs
    }

    // Take field
    pub fn take_pairs(&mut self) -> ::protobuf::RepeatedField<KvPair> {
        ::std::mem::replace(&mut self.pairs, ::protobuf::RepeatedField::new())
    }

    pub fn get_pairs(&self) -> &[KvPair] {
        &self.pairs
    }

    fn get_pairs_for_reflect(&self) -> &::protobuf::RepeatedField<KvPair> {
        &self.pairs
    }

    fn mut_pairs_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.pairs
    }
}

impl ::protobuf::Message for BatchGetResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.pairs {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.pairs)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        for value in &self.pairs {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        for v in &self.pairs {
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

impl ::protobuf::MessageStatic for BatchGetResponse {
    fn new() -> BatchGetResponse {
        BatchGetResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<BatchGetResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    BatchGetResponse::get_region_error_for_reflect,
                    BatchGetResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KvPair>>(
                    "pairs",
                    BatchGetResponse::get_pairs_for_reflect,
                    BatchGetResponse::mut_pairs_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<BatchGetResponse>(
                    "BatchGetResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for BatchGetResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_pairs();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for BatchGetResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for BatchGetResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ScanLockRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub max_version: u64,
    pub start_key: ::std::vec::Vec<u8>,
    pub limit: u32,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ScanLockRequest {}

impl ScanLockRequest {
    pub fn new() -> ScanLockRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ScanLockRequest {
        static mut instance: ::protobuf::lazy::Lazy<ScanLockRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ScanLockRequest,
        };
        unsafe {
            instance.get(ScanLockRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // uint64 max_version = 2;

    pub fn clear_max_version(&mut self) {
        self.max_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_max_version(&mut self, v: u64) {
        self.max_version = v;
    }

    pub fn get_max_version(&self) -> u64 {
        self.max_version
    }

    fn get_max_version_for_reflect(&self) -> &u64 {
        &self.max_version
    }

    fn mut_max_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.max_version
    }

    // bytes start_key = 3;

    pub fn clear_start_key(&mut self) {
        self.start_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_start_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.start_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_start_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // Take field
    pub fn take_start_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.start_key, ::std::vec::Vec::new())
    }

    pub fn get_start_key(&self) -> &[u8] {
        &self.start_key
    }

    fn get_start_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.start_key
    }

    fn mut_start_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // uint32 limit = 4;

    pub fn clear_limit(&mut self) {
        self.limit = 0;
    }

    // Param is passed by value, moved
    pub fn set_limit(&mut self, v: u32) {
        self.limit = v;
    }

    pub fn get_limit(&self) -> u32 {
        self.limit
    }

    fn get_limit_for_reflect(&self) -> &u32 {
        &self.limit
    }

    fn mut_limit_for_reflect(&mut self) -> &mut u32 {
        &mut self.limit
    }
}

impl ::protobuf::Message for ScanLockRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.max_version = tmp;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.start_key)?;
                },
                4 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint32()?;
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
        if let Some(ref v) = self.context.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if self.max_version != 0 {
            my_size += ::protobuf::rt::value_size(2, self.max_version, ::protobuf::wire_format::WireTypeVarint);
        }
        if !self.start_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.start_key);
        }
        if self.limit != 0 {
            my_size += ::protobuf::rt::value_size(4, self.limit, ::protobuf::wire_format::WireTypeVarint);
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
        if self.max_version != 0 {
            os.write_uint64(2, self.max_version)?;
        }
        if !self.start_key.is_empty() {
            os.write_bytes(3, &self.start_key)?;
        }
        if self.limit != 0 {
            os.write_uint32(4, self.limit)?;
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

impl ::protobuf::MessageStatic for ScanLockRequest {
    fn new() -> ScanLockRequest {
        ScanLockRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<ScanLockRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    ScanLockRequest::get_context_for_reflect,
                    ScanLockRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "max_version",
                    ScanLockRequest::get_max_version_for_reflect,
                    ScanLockRequest::mut_max_version_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "start_key",
                    ScanLockRequest::get_start_key_for_reflect,
                    ScanLockRequest::mut_start_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint32>(
                    "limit",
                    ScanLockRequest::get_limit_for_reflect,
                    ScanLockRequest::mut_limit_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ScanLockRequest>(
                    "ScanLockRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ScanLockRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_max_version();
        self.clear_start_key();
        self.clear_limit();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ScanLockRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ScanLockRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ScanLockResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::protobuf::SingularPtrField<KeyError>,
    pub locks: ::protobuf::RepeatedField<LockInfo>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ScanLockResponse {}

impl ScanLockResponse {
    pub fn new() -> ScanLockResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ScanLockResponse {
        static mut instance: ::protobuf::lazy::Lazy<ScanLockResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ScanLockResponse,
        };
        unsafe {
            instance.get(ScanLockResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // .kvrpcpb.KeyError error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: KeyError) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut KeyError {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> KeyError {
        self.error.take().unwrap_or_else(|| KeyError::new())
    }

    pub fn get_error(&self) -> &KeyError {
        self.error.as_ref().unwrap_or_else(|| KeyError::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<KeyError> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<KeyError> {
        &mut self.error
    }

    // repeated .kvrpcpb.LockInfo locks = 3;

    pub fn clear_locks(&mut self) {
        self.locks.clear();
    }

    // Param is passed by value, moved
    pub fn set_locks(&mut self, v: ::protobuf::RepeatedField<LockInfo>) {
        self.locks = v;
    }

    // Mutable pointer to the field.
    pub fn mut_locks(&mut self) -> &mut ::protobuf::RepeatedField<LockInfo> {
        &mut self.locks
    }

    // Take field
    pub fn take_locks(&mut self) -> ::protobuf::RepeatedField<LockInfo> {
        ::std::mem::replace(&mut self.locks, ::protobuf::RepeatedField::new())
    }

    pub fn get_locks(&self) -> &[LockInfo] {
        &self.locks
    }

    fn get_locks_for_reflect(&self) -> &::protobuf::RepeatedField<LockInfo> {
        &self.locks
    }

    fn mut_locks_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<LockInfo> {
        &mut self.locks
    }
}

impl ::protobuf::Message for ScanLockResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.error {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.locks {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.error)?;
                },
                3 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.locks)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        for value in &self.locks {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.error.as_ref() {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        for v in &self.locks {
            os.write_tag(3, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for ScanLockResponse {
    fn new() -> ScanLockResponse {
        ScanLockResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<ScanLockResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    ScanLockResponse::get_region_error_for_reflect,
                    ScanLockResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyError>>(
                    "error",
                    ScanLockResponse::get_error_for_reflect,
                    ScanLockResponse::mut_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<LockInfo>>(
                    "locks",
                    ScanLockResponse::get_locks_for_reflect,
                    ScanLockResponse::mut_locks_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ScanLockResponse>(
                    "ScanLockResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ScanLockResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.clear_locks();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ScanLockResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ScanLockResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct TxnInfo {
    // message fields
    pub txn: u64,
    pub status: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for TxnInfo {}

impl TxnInfo {
    pub fn new() -> TxnInfo {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static TxnInfo {
        static mut instance: ::protobuf::lazy::Lazy<TxnInfo> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const TxnInfo,
        };
        unsafe {
            instance.get(TxnInfo::new)
        }
    }

    // uint64 txn = 1;

    pub fn clear_txn(&mut self) {
        self.txn = 0;
    }

    // Param is passed by value, moved
    pub fn set_txn(&mut self, v: u64) {
        self.txn = v;
    }

    pub fn get_txn(&self) -> u64 {
        self.txn
    }

    fn get_txn_for_reflect(&self) -> &u64 {
        &self.txn
    }

    fn mut_txn_for_reflect(&mut self) -> &mut u64 {
        &mut self.txn
    }

    // uint64 status = 2;

    pub fn clear_status(&mut self) {
        self.status = 0;
    }

    // Param is passed by value, moved
    pub fn set_status(&mut self, v: u64) {
        self.status = v;
    }

    pub fn get_status(&self) -> u64 {
        self.status
    }

    fn get_status_for_reflect(&self) -> &u64 {
        &self.status
    }

    fn mut_status_for_reflect(&mut self) -> &mut u64 {
        &mut self.status
    }
}

impl ::protobuf::Message for TxnInfo {
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
                    self.txn = tmp;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.status = tmp;
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
        if self.txn != 0 {
            my_size += ::protobuf::rt::value_size(1, self.txn, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.status != 0 {
            my_size += ::protobuf::rt::value_size(2, self.status, ::protobuf::wire_format::WireTypeVarint);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.txn != 0 {
            os.write_uint64(1, self.txn)?;
        }
        if self.status != 0 {
            os.write_uint64(2, self.status)?;
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

impl ::protobuf::MessageStatic for TxnInfo {
    fn new() -> TxnInfo {
        TxnInfo::new()
    }

    fn descriptor_static(_: ::std::option::Option<TxnInfo>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "txn",
                    TxnInfo::get_txn_for_reflect,
                    TxnInfo::mut_txn_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "status",
                    TxnInfo::get_status_for_reflect,
                    TxnInfo::mut_status_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<TxnInfo>(
                    "TxnInfo",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for TxnInfo {
    fn clear(&mut self) {
        self.clear_txn();
        self.clear_status();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for TxnInfo {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for TxnInfo {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ResolveLockRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub start_version: u64,
    pub commit_version: u64,
    pub txn_infos: ::protobuf::RepeatedField<TxnInfo>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ResolveLockRequest {}

impl ResolveLockRequest {
    pub fn new() -> ResolveLockRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ResolveLockRequest {
        static mut instance: ::protobuf::lazy::Lazy<ResolveLockRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ResolveLockRequest,
        };
        unsafe {
            instance.get(ResolveLockRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // uint64 start_version = 2;

    pub fn clear_start_version(&mut self) {
        self.start_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_version(&mut self, v: u64) {
        self.start_version = v;
    }

    pub fn get_start_version(&self) -> u64 {
        self.start_version
    }

    fn get_start_version_for_reflect(&self) -> &u64 {
        &self.start_version
    }

    fn mut_start_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_version
    }

    // uint64 commit_version = 3;

    pub fn clear_commit_version(&mut self) {
        self.commit_version = 0;
    }

    // Param is passed by value, moved
    pub fn set_commit_version(&mut self, v: u64) {
        self.commit_version = v;
    }

    pub fn get_commit_version(&self) -> u64 {
        self.commit_version
    }

    fn get_commit_version_for_reflect(&self) -> &u64 {
        &self.commit_version
    }

    fn mut_commit_version_for_reflect(&mut self) -> &mut u64 {
        &mut self.commit_version
    }

    // repeated .kvrpcpb.TxnInfo txn_infos = 4;

    pub fn clear_txn_infos(&mut self) {
        self.txn_infos.clear();
    }

    // Param is passed by value, moved
    pub fn set_txn_infos(&mut self, v: ::protobuf::RepeatedField<TxnInfo>) {
        self.txn_infos = v;
    }

    // Mutable pointer to the field.
    pub fn mut_txn_infos(&mut self) -> &mut ::protobuf::RepeatedField<TxnInfo> {
        &mut self.txn_infos
    }

    // Take field
    pub fn take_txn_infos(&mut self) -> ::protobuf::RepeatedField<TxnInfo> {
        ::std::mem::replace(&mut self.txn_infos, ::protobuf::RepeatedField::new())
    }

    pub fn get_txn_infos(&self) -> &[TxnInfo] {
        &self.txn_infos
    }

    fn get_txn_infos_for_reflect(&self) -> &::protobuf::RepeatedField<TxnInfo> {
        &self.txn_infos
    }

    fn mut_txn_infos_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<TxnInfo> {
        &mut self.txn_infos
    }
}

impl ::protobuf::Message for ResolveLockRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.txn_infos {
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
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.start_version = tmp;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.commit_version = tmp;
                },
                4 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.txn_infos)?;
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
        if self.start_version != 0 {
            my_size += ::protobuf::rt::value_size(2, self.start_version, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.commit_version != 0 {
            my_size += ::protobuf::rt::value_size(3, self.commit_version, ::protobuf::wire_format::WireTypeVarint);
        }
        for value in &self.txn_infos {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
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
        if self.start_version != 0 {
            os.write_uint64(2, self.start_version)?;
        }
        if self.commit_version != 0 {
            os.write_uint64(3, self.commit_version)?;
        }
        for v in &self.txn_infos {
            os.write_tag(4, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for ResolveLockRequest {
    fn new() -> ResolveLockRequest {
        ResolveLockRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<ResolveLockRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    ResolveLockRequest::get_context_for_reflect,
                    ResolveLockRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_version",
                    ResolveLockRequest::get_start_version_for_reflect,
                    ResolveLockRequest::mut_start_version_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "commit_version",
                    ResolveLockRequest::get_commit_version_for_reflect,
                    ResolveLockRequest::mut_commit_version_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<TxnInfo>>(
                    "txn_infos",
                    ResolveLockRequest::get_txn_infos_for_reflect,
                    ResolveLockRequest::mut_txn_infos_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ResolveLockRequest>(
                    "ResolveLockRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ResolveLockRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_start_version();
        self.clear_commit_version();
        self.clear_txn_infos();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ResolveLockRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ResolveLockRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct ResolveLockResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::protobuf::SingularPtrField<KeyError>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for ResolveLockResponse {}

impl ResolveLockResponse {
    pub fn new() -> ResolveLockResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static ResolveLockResponse {
        static mut instance: ::protobuf::lazy::Lazy<ResolveLockResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ResolveLockResponse,
        };
        unsafe {
            instance.get(ResolveLockResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // .kvrpcpb.KeyError error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: KeyError) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut KeyError {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> KeyError {
        self.error.take().unwrap_or_else(|| KeyError::new())
    }

    pub fn get_error(&self) -> &KeyError {
        self.error.as_ref().unwrap_or_else(|| KeyError::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<KeyError> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<KeyError> {
        &mut self.error
    }
}

impl ::protobuf::Message for ResolveLockResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.error.as_ref() {
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

impl ::protobuf::MessageStatic for ResolveLockResponse {
    fn new() -> ResolveLockResponse {
        ResolveLockResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<ResolveLockResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    ResolveLockResponse::get_region_error_for_reflect,
                    ResolveLockResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyError>>(
                    "error",
                    ResolveLockResponse::get_error_for_reflect,
                    ResolveLockResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<ResolveLockResponse>(
                    "ResolveLockResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for ResolveLockResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for ResolveLockResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for ResolveLockResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct GCRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub safe_point: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for GCRequest {}

impl GCRequest {
    pub fn new() -> GCRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static GCRequest {
        static mut instance: ::protobuf::lazy::Lazy<GCRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const GCRequest,
        };
        unsafe {
            instance.get(GCRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // uint64 safe_point = 2;

    pub fn clear_safe_point(&mut self) {
        self.safe_point = 0;
    }

    // Param is passed by value, moved
    pub fn set_safe_point(&mut self, v: u64) {
        self.safe_point = v;
    }

    pub fn get_safe_point(&self) -> u64 {
        self.safe_point
    }

    fn get_safe_point_for_reflect(&self) -> &u64 {
        &self.safe_point
    }

    fn mut_safe_point_for_reflect(&mut self) -> &mut u64 {
        &mut self.safe_point
    }
}

impl ::protobuf::Message for GCRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.safe_point = tmp;
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
        if self.safe_point != 0 {
            my_size += ::protobuf::rt::value_size(2, self.safe_point, ::protobuf::wire_format::WireTypeVarint);
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
        if self.safe_point != 0 {
            os.write_uint64(2, self.safe_point)?;
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

impl ::protobuf::MessageStatic for GCRequest {
    fn new() -> GCRequest {
        GCRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<GCRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    GCRequest::get_context_for_reflect,
                    GCRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "safe_point",
                    GCRequest::get_safe_point_for_reflect,
                    GCRequest::mut_safe_point_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<GCRequest>(
                    "GCRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for GCRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_safe_point();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for GCRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for GCRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct GCResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::protobuf::SingularPtrField<KeyError>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for GCResponse {}

impl GCResponse {
    pub fn new() -> GCResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static GCResponse {
        static mut instance: ::protobuf::lazy::Lazy<GCResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const GCResponse,
        };
        unsafe {
            instance.get(GCResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // .kvrpcpb.KeyError error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    pub fn has_error(&self) -> bool {
        self.error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: KeyError) {
        self.error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut KeyError {
        if self.error.is_none() {
            self.error.set_default();
        }
        self.error.as_mut().unwrap()
    }

    // Take field
    pub fn take_error(&mut self) -> KeyError {
        self.error.take().unwrap_or_else(|| KeyError::new())
    }

    pub fn get_error(&self) -> &KeyError {
        self.error.as_ref().unwrap_or_else(|| KeyError::default_instance())
    }

    fn get_error_for_reflect(&self) -> &::protobuf::SingularPtrField<KeyError> {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<KeyError> {
        &mut self.error
    }
}

impl ::protobuf::Message for GCResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.error.as_ref() {
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

impl ::protobuf::MessageStatic for GCResponse {
    fn new() -> GCResponse {
        GCResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<GCResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    GCResponse::get_region_error_for_reflect,
                    GCResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyError>>(
                    "error",
                    GCResponse::get_error_for_reflect,
                    GCResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<GCResponse>(
                    "GCResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for GCResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for GCResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for GCResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawGetRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub key: ::std::vec::Vec<u8>,
    pub cf: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawGetRequest {}

impl RawGetRequest {
    pub fn new() -> RawGetRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawGetRequest {
        static mut instance: ::protobuf::lazy::Lazy<RawGetRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawGetRequest,
        };
        unsafe {
            instance.get(RawGetRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
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

    // string cf = 3;

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
}

impl ::protobuf::Message for RawGetRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
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
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.key);
        }
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(3, &self.cf);
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
        if !self.key.is_empty() {
            os.write_bytes(2, &self.key)?;
        }
        if !self.cf.is_empty() {
            os.write_string(3, &self.cf)?;
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

impl ::protobuf::MessageStatic for RawGetRequest {
    fn new() -> RawGetRequest {
        RawGetRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawGetRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    RawGetRequest::get_context_for_reflect,
                    RawGetRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    RawGetRequest::get_key_for_reflect,
                    RawGetRequest::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RawGetRequest::get_cf_for_reflect,
                    RawGetRequest::mut_cf_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawGetRequest>(
                    "RawGetRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawGetRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_key();
        self.clear_cf();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawGetRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawGetRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawGetResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    pub value: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawGetResponse {}

impl RawGetResponse {
    pub fn new() -> RawGetResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawGetResponse {
        static mut instance: ::protobuf::lazy::Lazy<RawGetResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawGetResponse,
        };
        unsafe {
            instance.get(RawGetResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
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

impl ::protobuf::Message for RawGetResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
        }
        if !self.value.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.value);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
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

impl ::protobuf::MessageStatic for RawGetResponse {
    fn new() -> RawGetResponse {
        RawGetResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawGetResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    RawGetResponse::get_region_error_for_reflect,
                    RawGetResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    RawGetResponse::get_error_for_reflect,
                    RawGetResponse::mut_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "value",
                    RawGetResponse::get_value_for_reflect,
                    RawGetResponse::mut_value_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawGetResponse>(
                    "RawGetResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawGetResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.clear_value();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawGetResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawGetResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawPutRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub key: ::std::vec::Vec<u8>,
    pub value: ::std::vec::Vec<u8>,
    pub cf: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawPutRequest {}

impl RawPutRequest {
    pub fn new() -> RawPutRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawPutRequest {
        static mut instance: ::protobuf::lazy::Lazy<RawPutRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawPutRequest,
        };
        unsafe {
            instance.get(RawPutRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
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

    // string cf = 4;

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
}

impl ::protobuf::Message for RawPutRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.value)?;
                },
                4 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
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
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.key);
        }
        if !self.value.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.value);
        }
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(4, &self.cf);
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
        if !self.key.is_empty() {
            os.write_bytes(2, &self.key)?;
        }
        if !self.value.is_empty() {
            os.write_bytes(3, &self.value)?;
        }
        if !self.cf.is_empty() {
            os.write_string(4, &self.cf)?;
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

impl ::protobuf::MessageStatic for RawPutRequest {
    fn new() -> RawPutRequest {
        RawPutRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawPutRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    RawPutRequest::get_context_for_reflect,
                    RawPutRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    RawPutRequest::get_key_for_reflect,
                    RawPutRequest::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "value",
                    RawPutRequest::get_value_for_reflect,
                    RawPutRequest::mut_value_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RawPutRequest::get_cf_for_reflect,
                    RawPutRequest::mut_cf_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawPutRequest>(
                    "RawPutRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawPutRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_key();
        self.clear_value();
        self.clear_cf();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawPutRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawPutRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawPutResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawPutResponse {}

impl RawPutResponse {
    pub fn new() -> RawPutResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawPutResponse {
        static mut instance: ::protobuf::lazy::Lazy<RawPutResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawPutResponse,
        };
        unsafe {
            instance.get(RawPutResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }
}

impl ::protobuf::Message for RawPutResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
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

impl ::protobuf::MessageStatic for RawPutResponse {
    fn new() -> RawPutResponse {
        RawPutResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawPutResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    RawPutResponse::get_region_error_for_reflect,
                    RawPutResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    RawPutResponse::get_error_for_reflect,
                    RawPutResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawPutResponse>(
                    "RawPutResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawPutResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawPutResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawPutResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawBatchPutRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub pairs: ::protobuf::RepeatedField<KvPair>,
    pub cf: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawBatchPutRequest {}

impl RawBatchPutRequest {
    pub fn new() -> RawBatchPutRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawBatchPutRequest {
        static mut instance: ::protobuf::lazy::Lazy<RawBatchPutRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawBatchPutRequest,
        };
        unsafe {
            instance.get(RawBatchPutRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // repeated .kvrpcpb.KvPair pairs = 2;

    pub fn clear_pairs(&mut self) {
        self.pairs.clear();
    }

    // Param is passed by value, moved
    pub fn set_pairs(&mut self, v: ::protobuf::RepeatedField<KvPair>) {
        self.pairs = v;
    }

    // Mutable pointer to the field.
    pub fn mut_pairs(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.pairs
    }

    // Take field
    pub fn take_pairs(&mut self) -> ::protobuf::RepeatedField<KvPair> {
        ::std::mem::replace(&mut self.pairs, ::protobuf::RepeatedField::new())
    }

    pub fn get_pairs(&self) -> &[KvPair] {
        &self.pairs
    }

    fn get_pairs_for_reflect(&self) -> &::protobuf::RepeatedField<KvPair> {
        &self.pairs
    }

    fn mut_pairs_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.pairs
    }

    // string cf = 3;

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
}

impl ::protobuf::Message for RawBatchPutRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.pairs {
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
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.pairs)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
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
        for value in &self.pairs {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(3, &self.cf);
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
        for v in &self.pairs {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        };
        if !self.cf.is_empty() {
            os.write_string(3, &self.cf)?;
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

impl ::protobuf::MessageStatic for RawBatchPutRequest {
    fn new() -> RawBatchPutRequest {
        RawBatchPutRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawBatchPutRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    RawBatchPutRequest::get_context_for_reflect,
                    RawBatchPutRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KvPair>>(
                    "pairs",
                    RawBatchPutRequest::get_pairs_for_reflect,
                    RawBatchPutRequest::mut_pairs_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RawBatchPutRequest::get_cf_for_reflect,
                    RawBatchPutRequest::mut_cf_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawBatchPutRequest>(
                    "RawBatchPutRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawBatchPutRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_pairs();
        self.clear_cf();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawBatchPutRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawBatchPutRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawBatchPutResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawBatchPutResponse {}

impl RawBatchPutResponse {
    pub fn new() -> RawBatchPutResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawBatchPutResponse {
        static mut instance: ::protobuf::lazy::Lazy<RawBatchPutResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawBatchPutResponse,
        };
        unsafe {
            instance.get(RawBatchPutResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }
}

impl ::protobuf::Message for RawBatchPutResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
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

impl ::protobuf::MessageStatic for RawBatchPutResponse {
    fn new() -> RawBatchPutResponse {
        RawBatchPutResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawBatchPutResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    RawBatchPutResponse::get_region_error_for_reflect,
                    RawBatchPutResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    RawBatchPutResponse::get_error_for_reflect,
                    RawBatchPutResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawBatchPutResponse>(
                    "RawBatchPutResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawBatchPutResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawBatchPutResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawBatchPutResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawBatchGetRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub keys: ::protobuf::RepeatedField<::std::vec::Vec<u8>>,
    pub cf: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawBatchGetRequest {}

impl RawBatchGetRequest {
    pub fn new() -> RawBatchGetRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawBatchGetRequest {
        static mut instance: ::protobuf::lazy::Lazy<RawBatchGetRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawBatchGetRequest,
        };
        unsafe {
            instance.get(RawBatchGetRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // repeated bytes keys = 2;

    pub fn clear_keys(&mut self) {
        self.keys.clear();
    }

    // Param is passed by value, moved
    pub fn set_keys(&mut self, v: ::protobuf::RepeatedField<::std::vec::Vec<u8>>) {
        self.keys = v;
    }

    // Mutable pointer to the field.
    pub fn mut_keys(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }

    // Take field
    pub fn take_keys(&mut self) -> ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        ::std::mem::replace(&mut self.keys, ::protobuf::RepeatedField::new())
    }

    pub fn get_keys(&self) -> &[::std::vec::Vec<u8>] {
        &self.keys
    }

    fn get_keys_for_reflect(&self) -> &::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &self.keys
    }

    fn mut_keys_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }

    // string cf = 3;

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
}

impl ::protobuf::Message for RawBatchGetRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_repeated_bytes_into(wire_type, is, &mut self.keys)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
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
        for value in &self.keys {
            my_size += ::protobuf::rt::bytes_size(2, &value);
        };
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(3, &self.cf);
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
        for v in &self.keys {
            os.write_bytes(2, &v)?;
        };
        if !self.cf.is_empty() {
            os.write_string(3, &self.cf)?;
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

impl ::protobuf::MessageStatic for RawBatchGetRequest {
    fn new() -> RawBatchGetRequest {
        RawBatchGetRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawBatchGetRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    RawBatchGetRequest::get_context_for_reflect,
                    RawBatchGetRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "keys",
                    RawBatchGetRequest::get_keys_for_reflect,
                    RawBatchGetRequest::mut_keys_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RawBatchGetRequest::get_cf_for_reflect,
                    RawBatchGetRequest::mut_cf_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawBatchGetRequest>(
                    "RawBatchGetRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawBatchGetRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_keys();
        self.clear_cf();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawBatchGetRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawBatchGetRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawBatchGetResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub pairs: ::protobuf::RepeatedField<KvPair>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawBatchGetResponse {}

impl RawBatchGetResponse {
    pub fn new() -> RawBatchGetResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawBatchGetResponse {
        static mut instance: ::protobuf::lazy::Lazy<RawBatchGetResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawBatchGetResponse,
        };
        unsafe {
            instance.get(RawBatchGetResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // repeated .kvrpcpb.KvPair pairs = 2;

    pub fn clear_pairs(&mut self) {
        self.pairs.clear();
    }

    // Param is passed by value, moved
    pub fn set_pairs(&mut self, v: ::protobuf::RepeatedField<KvPair>) {
        self.pairs = v;
    }

    // Mutable pointer to the field.
    pub fn mut_pairs(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.pairs
    }

    // Take field
    pub fn take_pairs(&mut self) -> ::protobuf::RepeatedField<KvPair> {
        ::std::mem::replace(&mut self.pairs, ::protobuf::RepeatedField::new())
    }

    pub fn get_pairs(&self) -> &[KvPair] {
        &self.pairs
    }

    fn get_pairs_for_reflect(&self) -> &::protobuf::RepeatedField<KvPair> {
        &self.pairs
    }

    fn mut_pairs_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.pairs
    }
}

impl ::protobuf::Message for RawBatchGetResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.pairs {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.pairs)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        for value in &self.pairs {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        for v in &self.pairs {
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

impl ::protobuf::MessageStatic for RawBatchGetResponse {
    fn new() -> RawBatchGetResponse {
        RawBatchGetResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawBatchGetResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    RawBatchGetResponse::get_region_error_for_reflect,
                    RawBatchGetResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KvPair>>(
                    "pairs",
                    RawBatchGetResponse::get_pairs_for_reflect,
                    RawBatchGetResponse::mut_pairs_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawBatchGetResponse>(
                    "RawBatchGetResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawBatchGetResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_pairs();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawBatchGetResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawBatchGetResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawDeleteRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub key: ::std::vec::Vec<u8>,
    pub cf: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawDeleteRequest {}

impl RawDeleteRequest {
    pub fn new() -> RawDeleteRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawDeleteRequest {
        static mut instance: ::protobuf::lazy::Lazy<RawDeleteRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawDeleteRequest,
        };
        unsafe {
            instance.get(RawDeleteRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
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

    // string cf = 3;

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
}

impl ::protobuf::Message for RawDeleteRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
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
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.key);
        }
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(3, &self.cf);
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
        if !self.key.is_empty() {
            os.write_bytes(2, &self.key)?;
        }
        if !self.cf.is_empty() {
            os.write_string(3, &self.cf)?;
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

impl ::protobuf::MessageStatic for RawDeleteRequest {
    fn new() -> RawDeleteRequest {
        RawDeleteRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawDeleteRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    RawDeleteRequest::get_context_for_reflect,
                    RawDeleteRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    RawDeleteRequest::get_key_for_reflect,
                    RawDeleteRequest::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RawDeleteRequest::get_cf_for_reflect,
                    RawDeleteRequest::mut_cf_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawDeleteRequest>(
                    "RawDeleteRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawDeleteRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_key();
        self.clear_cf();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawDeleteRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawDeleteRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawDeleteResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawDeleteResponse {}

impl RawDeleteResponse {
    pub fn new() -> RawDeleteResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawDeleteResponse {
        static mut instance: ::protobuf::lazy::Lazy<RawDeleteResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawDeleteResponse,
        };
        unsafe {
            instance.get(RawDeleteResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }
}

impl ::protobuf::Message for RawDeleteResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
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

impl ::protobuf::MessageStatic for RawDeleteResponse {
    fn new() -> RawDeleteResponse {
        RawDeleteResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawDeleteResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    RawDeleteResponse::get_region_error_for_reflect,
                    RawDeleteResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    RawDeleteResponse::get_error_for_reflect,
                    RawDeleteResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawDeleteResponse>(
                    "RawDeleteResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawDeleteResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawDeleteResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawDeleteResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawBatchDeleteRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub keys: ::protobuf::RepeatedField<::std::vec::Vec<u8>>,
    pub cf: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawBatchDeleteRequest {}

impl RawBatchDeleteRequest {
    pub fn new() -> RawBatchDeleteRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawBatchDeleteRequest {
        static mut instance: ::protobuf::lazy::Lazy<RawBatchDeleteRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawBatchDeleteRequest,
        };
        unsafe {
            instance.get(RawBatchDeleteRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // repeated bytes keys = 2;

    pub fn clear_keys(&mut self) {
        self.keys.clear();
    }

    // Param is passed by value, moved
    pub fn set_keys(&mut self, v: ::protobuf::RepeatedField<::std::vec::Vec<u8>>) {
        self.keys = v;
    }

    // Mutable pointer to the field.
    pub fn mut_keys(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }

    // Take field
    pub fn take_keys(&mut self) -> ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        ::std::mem::replace(&mut self.keys, ::protobuf::RepeatedField::new())
    }

    pub fn get_keys(&self) -> &[::std::vec::Vec<u8>] {
        &self.keys
    }

    fn get_keys_for_reflect(&self) -> &::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &self.keys
    }

    fn mut_keys_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<::std::vec::Vec<u8>> {
        &mut self.keys
    }

    // string cf = 3;

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
}

impl ::protobuf::Message for RawBatchDeleteRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_repeated_bytes_into(wire_type, is, &mut self.keys)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
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
        for value in &self.keys {
            my_size += ::protobuf::rt::bytes_size(2, &value);
        };
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(3, &self.cf);
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
        for v in &self.keys {
            os.write_bytes(2, &v)?;
        };
        if !self.cf.is_empty() {
            os.write_string(3, &self.cf)?;
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

impl ::protobuf::MessageStatic for RawBatchDeleteRequest {
    fn new() -> RawBatchDeleteRequest {
        RawBatchDeleteRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawBatchDeleteRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    RawBatchDeleteRequest::get_context_for_reflect,
                    RawBatchDeleteRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "keys",
                    RawBatchDeleteRequest::get_keys_for_reflect,
                    RawBatchDeleteRequest::mut_keys_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RawBatchDeleteRequest::get_cf_for_reflect,
                    RawBatchDeleteRequest::mut_cf_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawBatchDeleteRequest>(
                    "RawBatchDeleteRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawBatchDeleteRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_keys();
        self.clear_cf();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawBatchDeleteRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawBatchDeleteRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawBatchDeleteResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawBatchDeleteResponse {}

impl RawBatchDeleteResponse {
    pub fn new() -> RawBatchDeleteResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawBatchDeleteResponse {
        static mut instance: ::protobuf::lazy::Lazy<RawBatchDeleteResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawBatchDeleteResponse,
        };
        unsafe {
            instance.get(RawBatchDeleteResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }
}

impl ::protobuf::Message for RawBatchDeleteResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
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

impl ::protobuf::MessageStatic for RawBatchDeleteResponse {
    fn new() -> RawBatchDeleteResponse {
        RawBatchDeleteResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawBatchDeleteResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    RawBatchDeleteResponse::get_region_error_for_reflect,
                    RawBatchDeleteResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    RawBatchDeleteResponse::get_error_for_reflect,
                    RawBatchDeleteResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawBatchDeleteResponse>(
                    "RawBatchDeleteResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawBatchDeleteResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawBatchDeleteResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawBatchDeleteResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct DeleteRangeRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub start_key: ::std::vec::Vec<u8>,
    pub end_key: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for DeleteRangeRequest {}

impl DeleteRangeRequest {
    pub fn new() -> DeleteRangeRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static DeleteRangeRequest {
        static mut instance: ::protobuf::lazy::Lazy<DeleteRangeRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const DeleteRangeRequest,
        };
        unsafe {
            instance.get(DeleteRangeRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // bytes start_key = 2;

    pub fn clear_start_key(&mut self) {
        self.start_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_start_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.start_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_start_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // Take field
    pub fn take_start_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.start_key, ::std::vec::Vec::new())
    }

    pub fn get_start_key(&self) -> &[u8] {
        &self.start_key
    }

    fn get_start_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.start_key
    }

    fn mut_start_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // bytes end_key = 3;

    pub fn clear_end_key(&mut self) {
        self.end_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_end_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.end_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_end_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.end_key
    }

    // Take field
    pub fn take_end_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.end_key, ::std::vec::Vec::new())
    }

    pub fn get_end_key(&self) -> &[u8] {
        &self.end_key
    }

    fn get_end_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.end_key
    }

    fn mut_end_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.end_key
    }
}

impl ::protobuf::Message for DeleteRangeRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.start_key)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.end_key)?;
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
        if !self.start_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.start_key);
        }
        if !self.end_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.end_key);
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
        if !self.start_key.is_empty() {
            os.write_bytes(2, &self.start_key)?;
        }
        if !self.end_key.is_empty() {
            os.write_bytes(3, &self.end_key)?;
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

impl ::protobuf::MessageStatic for DeleteRangeRequest {
    fn new() -> DeleteRangeRequest {
        DeleteRangeRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<DeleteRangeRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    DeleteRangeRequest::get_context_for_reflect,
                    DeleteRangeRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "start_key",
                    DeleteRangeRequest::get_start_key_for_reflect,
                    DeleteRangeRequest::mut_start_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "end_key",
                    DeleteRangeRequest::get_end_key_for_reflect,
                    DeleteRangeRequest::mut_end_key_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<DeleteRangeRequest>(
                    "DeleteRangeRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for DeleteRangeRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_start_key();
        self.clear_end_key();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for DeleteRangeRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for DeleteRangeRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct DeleteRangeResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for DeleteRangeResponse {}

impl DeleteRangeResponse {
    pub fn new() -> DeleteRangeResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static DeleteRangeResponse {
        static mut instance: ::protobuf::lazy::Lazy<DeleteRangeResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const DeleteRangeResponse,
        };
        unsafe {
            instance.get(DeleteRangeResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }
}

impl ::protobuf::Message for DeleteRangeResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
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

impl ::protobuf::MessageStatic for DeleteRangeResponse {
    fn new() -> DeleteRangeResponse {
        DeleteRangeResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<DeleteRangeResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    DeleteRangeResponse::get_region_error_for_reflect,
                    DeleteRangeResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    DeleteRangeResponse::get_error_for_reflect,
                    DeleteRangeResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<DeleteRangeResponse>(
                    "DeleteRangeResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for DeleteRangeResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for DeleteRangeResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for DeleteRangeResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawDeleteRangeRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub start_key: ::std::vec::Vec<u8>,
    pub end_key: ::std::vec::Vec<u8>,
    pub cf: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawDeleteRangeRequest {}

impl RawDeleteRangeRequest {
    pub fn new() -> RawDeleteRangeRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawDeleteRangeRequest {
        static mut instance: ::protobuf::lazy::Lazy<RawDeleteRangeRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawDeleteRangeRequest,
        };
        unsafe {
            instance.get(RawDeleteRangeRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // bytes start_key = 2;

    pub fn clear_start_key(&mut self) {
        self.start_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_start_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.start_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_start_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // Take field
    pub fn take_start_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.start_key, ::std::vec::Vec::new())
    }

    pub fn get_start_key(&self) -> &[u8] {
        &self.start_key
    }

    fn get_start_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.start_key
    }

    fn mut_start_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // bytes end_key = 3;

    pub fn clear_end_key(&mut self) {
        self.end_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_end_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.end_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_end_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.end_key
    }

    // Take field
    pub fn take_end_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.end_key, ::std::vec::Vec::new())
    }

    pub fn get_end_key(&self) -> &[u8] {
        &self.end_key
    }

    fn get_end_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.end_key
    }

    fn mut_end_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.end_key
    }

    // string cf = 4;

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
}

impl ::protobuf::Message for RawDeleteRangeRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.start_key)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.end_key)?;
                },
                4 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
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
        if !self.start_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.start_key);
        }
        if !self.end_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.end_key);
        }
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(4, &self.cf);
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
        if !self.start_key.is_empty() {
            os.write_bytes(2, &self.start_key)?;
        }
        if !self.end_key.is_empty() {
            os.write_bytes(3, &self.end_key)?;
        }
        if !self.cf.is_empty() {
            os.write_string(4, &self.cf)?;
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

impl ::protobuf::MessageStatic for RawDeleteRangeRequest {
    fn new() -> RawDeleteRangeRequest {
        RawDeleteRangeRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawDeleteRangeRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    RawDeleteRangeRequest::get_context_for_reflect,
                    RawDeleteRangeRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "start_key",
                    RawDeleteRangeRequest::get_start_key_for_reflect,
                    RawDeleteRangeRequest::mut_start_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "end_key",
                    RawDeleteRangeRequest::get_end_key_for_reflect,
                    RawDeleteRangeRequest::mut_end_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RawDeleteRangeRequest::get_cf_for_reflect,
                    RawDeleteRangeRequest::mut_cf_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawDeleteRangeRequest>(
                    "RawDeleteRangeRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawDeleteRangeRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_start_key();
        self.clear_end_key();
        self.clear_cf();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawDeleteRangeRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawDeleteRangeRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawDeleteRangeResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawDeleteRangeResponse {}

impl RawDeleteRangeResponse {
    pub fn new() -> RawDeleteRangeResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawDeleteRangeResponse {
        static mut instance: ::protobuf::lazy::Lazy<RawDeleteRangeResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawDeleteRangeResponse,
        };
        unsafe {
            instance.get(RawDeleteRangeResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }
}

impl ::protobuf::Message for RawDeleteRangeResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
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

impl ::protobuf::MessageStatic for RawDeleteRangeResponse {
    fn new() -> RawDeleteRangeResponse {
        RawDeleteRangeResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawDeleteRangeResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    RawDeleteRangeResponse::get_region_error_for_reflect,
                    RawDeleteRangeResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    RawDeleteRangeResponse::get_error_for_reflect,
                    RawDeleteRangeResponse::mut_error_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawDeleteRangeResponse>(
                    "RawDeleteRangeResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawDeleteRangeResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawDeleteRangeResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawDeleteRangeResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawScanRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub start_key: ::std::vec::Vec<u8>,
    pub limit: u32,
    pub key_only: bool,
    pub cf: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawScanRequest {}

impl RawScanRequest {
    pub fn new() -> RawScanRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawScanRequest {
        static mut instance: ::protobuf::lazy::Lazy<RawScanRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawScanRequest,
        };
        unsafe {
            instance.get(RawScanRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // bytes start_key = 2;

    pub fn clear_start_key(&mut self) {
        self.start_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_start_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.start_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_start_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // Take field
    pub fn take_start_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.start_key, ::std::vec::Vec::new())
    }

    pub fn get_start_key(&self) -> &[u8] {
        &self.start_key
    }

    fn get_start_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.start_key
    }

    fn mut_start_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // uint32 limit = 3;

    pub fn clear_limit(&mut self) {
        self.limit = 0;
    }

    // Param is passed by value, moved
    pub fn set_limit(&mut self, v: u32) {
        self.limit = v;
    }

    pub fn get_limit(&self) -> u32 {
        self.limit
    }

    fn get_limit_for_reflect(&self) -> &u32 {
        &self.limit
    }

    fn mut_limit_for_reflect(&mut self) -> &mut u32 {
        &mut self.limit
    }

    // bool key_only = 4;

    pub fn clear_key_only(&mut self) {
        self.key_only = false;
    }

    // Param is passed by value, moved
    pub fn set_key_only(&mut self, v: bool) {
        self.key_only = v;
    }

    pub fn get_key_only(&self) -> bool {
        self.key_only
    }

    fn get_key_only_for_reflect(&self) -> &bool {
        &self.key_only
    }

    fn mut_key_only_for_reflect(&mut self) -> &mut bool {
        &mut self.key_only
    }

    // string cf = 5;

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
}

impl ::protobuf::Message for RawScanRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.start_key)?;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint32()?;
                    self.limit = tmp;
                },
                4 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_bool()?;
                    self.key_only = tmp;
                },
                5 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
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
        if !self.start_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.start_key);
        }
        if self.limit != 0 {
            my_size += ::protobuf::rt::value_size(3, self.limit, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.key_only != false {
            my_size += 2;
        }
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(5, &self.cf);
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
        if !self.start_key.is_empty() {
            os.write_bytes(2, &self.start_key)?;
        }
        if self.limit != 0 {
            os.write_uint32(3, self.limit)?;
        }
        if self.key_only != false {
            os.write_bool(4, self.key_only)?;
        }
        if !self.cf.is_empty() {
            os.write_string(5, &self.cf)?;
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

impl ::protobuf::MessageStatic for RawScanRequest {
    fn new() -> RawScanRequest {
        RawScanRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawScanRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    RawScanRequest::get_context_for_reflect,
                    RawScanRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "start_key",
                    RawScanRequest::get_start_key_for_reflect,
                    RawScanRequest::mut_start_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint32>(
                    "limit",
                    RawScanRequest::get_limit_for_reflect,
                    RawScanRequest::mut_limit_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "key_only",
                    RawScanRequest::get_key_only_for_reflect,
                    RawScanRequest::mut_key_only_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RawScanRequest::get_cf_for_reflect,
                    RawScanRequest::mut_cf_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawScanRequest>(
                    "RawScanRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawScanRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_start_key();
        self.clear_limit();
        self.clear_key_only();
        self.clear_cf();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawScanRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawScanRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawScanResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub kvs: ::protobuf::RepeatedField<KvPair>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawScanResponse {}

impl RawScanResponse {
    pub fn new() -> RawScanResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawScanResponse {
        static mut instance: ::protobuf::lazy::Lazy<RawScanResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawScanResponse,
        };
        unsafe {
            instance.get(RawScanResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // repeated .kvrpcpb.KvPair kvs = 2;

    pub fn clear_kvs(&mut self) {
        self.kvs.clear();
    }

    // Param is passed by value, moved
    pub fn set_kvs(&mut self, v: ::protobuf::RepeatedField<KvPair>) {
        self.kvs = v;
    }

    // Mutable pointer to the field.
    pub fn mut_kvs(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.kvs
    }

    // Take field
    pub fn take_kvs(&mut self) -> ::protobuf::RepeatedField<KvPair> {
        ::std::mem::replace(&mut self.kvs, ::protobuf::RepeatedField::new())
    }

    pub fn get_kvs(&self) -> &[KvPair] {
        &self.kvs
    }

    fn get_kvs_for_reflect(&self) -> &::protobuf::RepeatedField<KvPair> {
        &self.kvs
    }

    fn mut_kvs_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.kvs
    }
}

impl ::protobuf::Message for RawScanResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.kvs {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.kvs)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        for value in &self.kvs {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        for v in &self.kvs {
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

impl ::protobuf::MessageStatic for RawScanResponse {
    fn new() -> RawScanResponse {
        RawScanResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawScanResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    RawScanResponse::get_region_error_for_reflect,
                    RawScanResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KvPair>>(
                    "kvs",
                    RawScanResponse::get_kvs_for_reflect,
                    RawScanResponse::mut_kvs_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawScanResponse>(
                    "RawScanResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawScanResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_kvs();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawScanResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawScanResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct KeyRange {
    // message fields
    pub start_key: ::std::vec::Vec<u8>,
    pub end_key: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for KeyRange {}

impl KeyRange {
    pub fn new() -> KeyRange {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static KeyRange {
        static mut instance: ::protobuf::lazy::Lazy<KeyRange> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const KeyRange,
        };
        unsafe {
            instance.get(KeyRange::new)
        }
    }

    // bytes start_key = 1;

    pub fn clear_start_key(&mut self) {
        self.start_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_start_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.start_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_start_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // Take field
    pub fn take_start_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.start_key, ::std::vec::Vec::new())
    }

    pub fn get_start_key(&self) -> &[u8] {
        &self.start_key
    }

    fn get_start_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.start_key
    }

    fn mut_start_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.start_key
    }

    // bytes end_key = 2;

    pub fn clear_end_key(&mut self) {
        self.end_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_end_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.end_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_end_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.end_key
    }

    // Take field
    pub fn take_end_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.end_key, ::std::vec::Vec::new())
    }

    pub fn get_end_key(&self) -> &[u8] {
        &self.end_key
    }

    fn get_end_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.end_key
    }

    fn mut_end_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.end_key
    }
}

impl ::protobuf::Message for KeyRange {
    fn is_initialized(&self) -> bool {
        true
    }

    fn merge_from(&mut self, is: &mut ::protobuf::CodedInputStream) -> ::protobuf::ProtobufResult<()> {
        while !is.eof()? {
            let (field_number, wire_type) = is.read_tag_unpack()?;
            match field_number {
                1 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.start_key)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.end_key)?;
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
        if !self.start_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(1, &self.start_key);
        }
        if !self.end_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.end_key);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if !self.start_key.is_empty() {
            os.write_bytes(1, &self.start_key)?;
        }
        if !self.end_key.is_empty() {
            os.write_bytes(2, &self.end_key)?;
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

impl ::protobuf::MessageStatic for KeyRange {
    fn new() -> KeyRange {
        KeyRange::new()
    }

    fn descriptor_static(_: ::std::option::Option<KeyRange>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "start_key",
                    KeyRange::get_start_key_for_reflect,
                    KeyRange::mut_start_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "end_key",
                    KeyRange::get_end_key_for_reflect,
                    KeyRange::mut_end_key_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<KeyRange>(
                    "KeyRange",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for KeyRange {
    fn clear(&mut self) {
        self.clear_start_key();
        self.clear_end_key();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for KeyRange {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for KeyRange {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawBatchScanRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub ranges: ::protobuf::RepeatedField<KeyRange>,
    pub each_limit: u32,
    pub key_only: bool,
    pub cf: ::std::string::String,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawBatchScanRequest {}

impl RawBatchScanRequest {
    pub fn new() -> RawBatchScanRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawBatchScanRequest {
        static mut instance: ::protobuf::lazy::Lazy<RawBatchScanRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawBatchScanRequest,
        };
        unsafe {
            instance.get(RawBatchScanRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // repeated .kvrpcpb.KeyRange ranges = 2;

    pub fn clear_ranges(&mut self) {
        self.ranges.clear();
    }

    // Param is passed by value, moved
    pub fn set_ranges(&mut self, v: ::protobuf::RepeatedField<KeyRange>) {
        self.ranges = v;
    }

    // Mutable pointer to the field.
    pub fn mut_ranges(&mut self) -> &mut ::protobuf::RepeatedField<KeyRange> {
        &mut self.ranges
    }

    // Take field
    pub fn take_ranges(&mut self) -> ::protobuf::RepeatedField<KeyRange> {
        ::std::mem::replace(&mut self.ranges, ::protobuf::RepeatedField::new())
    }

    pub fn get_ranges(&self) -> &[KeyRange] {
        &self.ranges
    }

    fn get_ranges_for_reflect(&self) -> &::protobuf::RepeatedField<KeyRange> {
        &self.ranges
    }

    fn mut_ranges_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<KeyRange> {
        &mut self.ranges
    }

    // uint32 each_limit = 3;

    pub fn clear_each_limit(&mut self) {
        self.each_limit = 0;
    }

    // Param is passed by value, moved
    pub fn set_each_limit(&mut self, v: u32) {
        self.each_limit = v;
    }

    pub fn get_each_limit(&self) -> u32 {
        self.each_limit
    }

    fn get_each_limit_for_reflect(&self) -> &u32 {
        &self.each_limit
    }

    fn mut_each_limit_for_reflect(&mut self) -> &mut u32 {
        &mut self.each_limit
    }

    // bool key_only = 4;

    pub fn clear_key_only(&mut self) {
        self.key_only = false;
    }

    // Param is passed by value, moved
    pub fn set_key_only(&mut self, v: bool) {
        self.key_only = v;
    }

    pub fn get_key_only(&self) -> bool {
        self.key_only
    }

    fn get_key_only_for_reflect(&self) -> &bool {
        &self.key_only
    }

    fn mut_key_only_for_reflect(&mut self) -> &mut bool {
        &mut self.key_only
    }

    // string cf = 5;

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
}

impl ::protobuf::Message for RawBatchScanRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.ranges {
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
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.ranges)?;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint32()?;
                    self.each_limit = tmp;
                },
                4 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_bool()?;
                    self.key_only = tmp;
                },
                5 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.cf)?;
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
        for value in &self.ranges {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        if self.each_limit != 0 {
            my_size += ::protobuf::rt::value_size(3, self.each_limit, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.key_only != false {
            my_size += 2;
        }
        if !self.cf.is_empty() {
            my_size += ::protobuf::rt::string_size(5, &self.cf);
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
        for v in &self.ranges {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        };
        if self.each_limit != 0 {
            os.write_uint32(3, self.each_limit)?;
        }
        if self.key_only != false {
            os.write_bool(4, self.key_only)?;
        }
        if !self.cf.is_empty() {
            os.write_string(5, &self.cf)?;
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

impl ::protobuf::MessageStatic for RawBatchScanRequest {
    fn new() -> RawBatchScanRequest {
        RawBatchScanRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawBatchScanRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    RawBatchScanRequest::get_context_for_reflect,
                    RawBatchScanRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KeyRange>>(
                    "ranges",
                    RawBatchScanRequest::get_ranges_for_reflect,
                    RawBatchScanRequest::mut_ranges_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint32>(
                    "each_limit",
                    RawBatchScanRequest::get_each_limit_for_reflect,
                    RawBatchScanRequest::mut_each_limit_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBool>(
                    "key_only",
                    RawBatchScanRequest::get_key_only_for_reflect,
                    RawBatchScanRequest::mut_key_only_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "cf",
                    RawBatchScanRequest::get_cf_for_reflect,
                    RawBatchScanRequest::mut_cf_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawBatchScanRequest>(
                    "RawBatchScanRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawBatchScanRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_ranges();
        self.clear_each_limit();
        self.clear_key_only();
        self.clear_cf();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawBatchScanRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawBatchScanRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct RawBatchScanResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub kvs: ::protobuf::RepeatedField<KvPair>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for RawBatchScanResponse {}

impl RawBatchScanResponse {
    pub fn new() -> RawBatchScanResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static RawBatchScanResponse {
        static mut instance: ::protobuf::lazy::Lazy<RawBatchScanResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const RawBatchScanResponse,
        };
        unsafe {
            instance.get(RawBatchScanResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // repeated .kvrpcpb.KvPair kvs = 2;

    pub fn clear_kvs(&mut self) {
        self.kvs.clear();
    }

    // Param is passed by value, moved
    pub fn set_kvs(&mut self, v: ::protobuf::RepeatedField<KvPair>) {
        self.kvs = v;
    }

    // Mutable pointer to the field.
    pub fn mut_kvs(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.kvs
    }

    // Take field
    pub fn take_kvs(&mut self) -> ::protobuf::RepeatedField<KvPair> {
        ::std::mem::replace(&mut self.kvs, ::protobuf::RepeatedField::new())
    }

    pub fn get_kvs(&self) -> &[KvPair] {
        &self.kvs
    }

    fn get_kvs_for_reflect(&self) -> &::protobuf::RepeatedField<KvPair> {
        &self.kvs
    }

    fn mut_kvs_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<KvPair> {
        &mut self.kvs
    }
}

impl ::protobuf::Message for RawBatchScanResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.kvs {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.kvs)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        for value in &self.kvs {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        for v in &self.kvs {
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

impl ::protobuf::MessageStatic for RawBatchScanResponse {
    fn new() -> RawBatchScanResponse {
        RawBatchScanResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<RawBatchScanResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    RawBatchScanResponse::get_region_error_for_reflect,
                    RawBatchScanResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<KvPair>>(
                    "kvs",
                    RawBatchScanResponse::get_kvs_for_reflect,
                    RawBatchScanResponse::mut_kvs_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<RawBatchScanResponse>(
                    "RawBatchScanResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for RawBatchScanResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_kvs();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for RawBatchScanResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for RawBatchScanResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct MvccWrite {
    // message fields
    pub field_type: Op,
    pub start_ts: u64,
    pub commit_ts: u64,
    pub short_value: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for MvccWrite {}

impl MvccWrite {
    pub fn new() -> MvccWrite {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static MvccWrite {
        static mut instance: ::protobuf::lazy::Lazy<MvccWrite> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const MvccWrite,
        };
        unsafe {
            instance.get(MvccWrite::new)
        }
    }

    // .kvrpcpb.Op type = 1;

    pub fn clear_field_type(&mut self) {
        self.field_type = Op::Put;
    }

    // Param is passed by value, moved
    pub fn set_field_type(&mut self, v: Op) {
        self.field_type = v;
    }

    pub fn get_field_type(&self) -> Op {
        self.field_type
    }

    fn get_field_type_for_reflect(&self) -> &Op {
        &self.field_type
    }

    fn mut_field_type_for_reflect(&mut self) -> &mut Op {
        &mut self.field_type
    }

    // uint64 start_ts = 2;

    pub fn clear_start_ts(&mut self) {
        self.start_ts = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_ts(&mut self, v: u64) {
        self.start_ts = v;
    }

    pub fn get_start_ts(&self) -> u64 {
        self.start_ts
    }

    fn get_start_ts_for_reflect(&self) -> &u64 {
        &self.start_ts
    }

    fn mut_start_ts_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_ts
    }

    // uint64 commit_ts = 3;

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

    // bytes short_value = 4;

    pub fn clear_short_value(&mut self) {
        self.short_value.clear();
    }

    // Param is passed by value, moved
    pub fn set_short_value(&mut self, v: ::std::vec::Vec<u8>) {
        self.short_value = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_short_value(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.short_value
    }

    // Take field
    pub fn take_short_value(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.short_value, ::std::vec::Vec::new())
    }

    pub fn get_short_value(&self) -> &[u8] {
        &self.short_value
    }

    fn get_short_value_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.short_value
    }

    fn mut_short_value_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.short_value
    }
}

impl ::protobuf::Message for MvccWrite {
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
                    self.field_type = tmp;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.start_ts = tmp;
                },
                3 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.commit_ts = tmp;
                },
                4 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.short_value)?;
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
        if self.field_type != Op::Put {
            my_size += ::protobuf::rt::enum_size(1, self.field_type);
        }
        if self.start_ts != 0 {
            my_size += ::protobuf::rt::value_size(2, self.start_ts, ::protobuf::wire_format::WireTypeVarint);
        }
        if self.commit_ts != 0 {
            my_size += ::protobuf::rt::value_size(3, self.commit_ts, ::protobuf::wire_format::WireTypeVarint);
        }
        if !self.short_value.is_empty() {
            my_size += ::protobuf::rt::bytes_size(4, &self.short_value);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.field_type != Op::Put {
            os.write_enum(1, self.field_type.value())?;
        }
        if self.start_ts != 0 {
            os.write_uint64(2, self.start_ts)?;
        }
        if self.commit_ts != 0 {
            os.write_uint64(3, self.commit_ts)?;
        }
        if !self.short_value.is_empty() {
            os.write_bytes(4, &self.short_value)?;
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

impl ::protobuf::MessageStatic for MvccWrite {
    fn new() -> MvccWrite {
        MvccWrite::new()
    }

    fn descriptor_static(_: ::std::option::Option<MvccWrite>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeEnum<Op>>(
                    "type",
                    MvccWrite::get_field_type_for_reflect,
                    MvccWrite::mut_field_type_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_ts",
                    MvccWrite::get_start_ts_for_reflect,
                    MvccWrite::mut_start_ts_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "commit_ts",
                    MvccWrite::get_commit_ts_for_reflect,
                    MvccWrite::mut_commit_ts_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "short_value",
                    MvccWrite::get_short_value_for_reflect,
                    MvccWrite::mut_short_value_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<MvccWrite>(
                    "MvccWrite",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for MvccWrite {
    fn clear(&mut self) {
        self.clear_field_type();
        self.clear_start_ts();
        self.clear_commit_ts();
        self.clear_short_value();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for MvccWrite {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for MvccWrite {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct MvccValue {
    // message fields
    pub start_ts: u64,
    pub value: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for MvccValue {}

impl MvccValue {
    pub fn new() -> MvccValue {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static MvccValue {
        static mut instance: ::protobuf::lazy::Lazy<MvccValue> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const MvccValue,
        };
        unsafe {
            instance.get(MvccValue::new)
        }
    }

    // uint64 start_ts = 1;

    pub fn clear_start_ts(&mut self) {
        self.start_ts = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_ts(&mut self, v: u64) {
        self.start_ts = v;
    }

    pub fn get_start_ts(&self) -> u64 {
        self.start_ts
    }

    fn get_start_ts_for_reflect(&self) -> &u64 {
        &self.start_ts
    }

    fn mut_start_ts_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_ts
    }

    // bytes value = 2;

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

impl ::protobuf::Message for MvccValue {
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
                    self.start_ts = tmp;
                },
                2 => {
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
        if self.start_ts != 0 {
            my_size += ::protobuf::rt::value_size(1, self.start_ts, ::protobuf::wire_format::WireTypeVarint);
        }
        if !self.value.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.value);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.start_ts != 0 {
            os.write_uint64(1, self.start_ts)?;
        }
        if !self.value.is_empty() {
            os.write_bytes(2, &self.value)?;
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

impl ::protobuf::MessageStatic for MvccValue {
    fn new() -> MvccValue {
        MvccValue::new()
    }

    fn descriptor_static(_: ::std::option::Option<MvccValue>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_ts",
                    MvccValue::get_start_ts_for_reflect,
                    MvccValue::mut_start_ts_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "value",
                    MvccValue::get_value_for_reflect,
                    MvccValue::mut_value_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<MvccValue>(
                    "MvccValue",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for MvccValue {
    fn clear(&mut self) {
        self.clear_start_ts();
        self.clear_value();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for MvccValue {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for MvccValue {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct MvccLock {
    // message fields
    pub field_type: Op,
    pub start_ts: u64,
    pub primary: ::std::vec::Vec<u8>,
    pub short_value: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for MvccLock {}

impl MvccLock {
    pub fn new() -> MvccLock {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static MvccLock {
        static mut instance: ::protobuf::lazy::Lazy<MvccLock> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const MvccLock,
        };
        unsafe {
            instance.get(MvccLock::new)
        }
    }

    // .kvrpcpb.Op type = 1;

    pub fn clear_field_type(&mut self) {
        self.field_type = Op::Put;
    }

    // Param is passed by value, moved
    pub fn set_field_type(&mut self, v: Op) {
        self.field_type = v;
    }

    pub fn get_field_type(&self) -> Op {
        self.field_type
    }

    fn get_field_type_for_reflect(&self) -> &Op {
        &self.field_type
    }

    fn mut_field_type_for_reflect(&mut self) -> &mut Op {
        &mut self.field_type
    }

    // uint64 start_ts = 2;

    pub fn clear_start_ts(&mut self) {
        self.start_ts = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_ts(&mut self, v: u64) {
        self.start_ts = v;
    }

    pub fn get_start_ts(&self) -> u64 {
        self.start_ts
    }

    fn get_start_ts_for_reflect(&self) -> &u64 {
        &self.start_ts
    }

    fn mut_start_ts_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_ts
    }

    // bytes primary = 3;

    pub fn clear_primary(&mut self) {
        self.primary.clear();
    }

    // Param is passed by value, moved
    pub fn set_primary(&mut self, v: ::std::vec::Vec<u8>) {
        self.primary = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_primary(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.primary
    }

    // Take field
    pub fn take_primary(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.primary, ::std::vec::Vec::new())
    }

    pub fn get_primary(&self) -> &[u8] {
        &self.primary
    }

    fn get_primary_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.primary
    }

    fn mut_primary_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.primary
    }

    // bytes short_value = 4;

    pub fn clear_short_value(&mut self) {
        self.short_value.clear();
    }

    // Param is passed by value, moved
    pub fn set_short_value(&mut self, v: ::std::vec::Vec<u8>) {
        self.short_value = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_short_value(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.short_value
    }

    // Take field
    pub fn take_short_value(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.short_value, ::std::vec::Vec::new())
    }

    pub fn get_short_value(&self) -> &[u8] {
        &self.short_value
    }

    fn get_short_value_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.short_value
    }

    fn mut_short_value_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.short_value
    }
}

impl ::protobuf::Message for MvccLock {
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
                    self.field_type = tmp;
                },
                2 => {
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.start_ts = tmp;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.primary)?;
                },
                4 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.short_value)?;
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
        if self.field_type != Op::Put {
            my_size += ::protobuf::rt::enum_size(1, self.field_type);
        }
        if self.start_ts != 0 {
            my_size += ::protobuf::rt::value_size(2, self.start_ts, ::protobuf::wire_format::WireTypeVarint);
        }
        if !self.primary.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.primary);
        }
        if !self.short_value.is_empty() {
            my_size += ::protobuf::rt::bytes_size(4, &self.short_value);
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if self.field_type != Op::Put {
            os.write_enum(1, self.field_type.value())?;
        }
        if self.start_ts != 0 {
            os.write_uint64(2, self.start_ts)?;
        }
        if !self.primary.is_empty() {
            os.write_bytes(3, &self.primary)?;
        }
        if !self.short_value.is_empty() {
            os.write_bytes(4, &self.short_value)?;
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

impl ::protobuf::MessageStatic for MvccLock {
    fn new() -> MvccLock {
        MvccLock::new()
    }

    fn descriptor_static(_: ::std::option::Option<MvccLock>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeEnum<Op>>(
                    "type",
                    MvccLock::get_field_type_for_reflect,
                    MvccLock::mut_field_type_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_ts",
                    MvccLock::get_start_ts_for_reflect,
                    MvccLock::mut_start_ts_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "primary",
                    MvccLock::get_primary_for_reflect,
                    MvccLock::mut_primary_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "short_value",
                    MvccLock::get_short_value_for_reflect,
                    MvccLock::mut_short_value_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<MvccLock>(
                    "MvccLock",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for MvccLock {
    fn clear(&mut self) {
        self.clear_field_type();
        self.clear_start_ts();
        self.clear_primary();
        self.clear_short_value();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for MvccLock {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for MvccLock {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct MvccInfo {
    // message fields
    pub lock: ::protobuf::SingularPtrField<MvccLock>,
    pub writes: ::protobuf::RepeatedField<MvccWrite>,
    pub values: ::protobuf::RepeatedField<MvccValue>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for MvccInfo {}

impl MvccInfo {
    pub fn new() -> MvccInfo {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static MvccInfo {
        static mut instance: ::protobuf::lazy::Lazy<MvccInfo> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const MvccInfo,
        };
        unsafe {
            instance.get(MvccInfo::new)
        }
    }

    // .kvrpcpb.MvccLock lock = 1;

    pub fn clear_lock(&mut self) {
        self.lock.clear();
    }

    pub fn has_lock(&self) -> bool {
        self.lock.is_some()
    }

    // Param is passed by value, moved
    pub fn set_lock(&mut self, v: MvccLock) {
        self.lock = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_lock(&mut self) -> &mut MvccLock {
        if self.lock.is_none() {
            self.lock.set_default();
        }
        self.lock.as_mut().unwrap()
    }

    // Take field
    pub fn take_lock(&mut self) -> MvccLock {
        self.lock.take().unwrap_or_else(|| MvccLock::new())
    }

    pub fn get_lock(&self) -> &MvccLock {
        self.lock.as_ref().unwrap_or_else(|| MvccLock::default_instance())
    }

    fn get_lock_for_reflect(&self) -> &::protobuf::SingularPtrField<MvccLock> {
        &self.lock
    }

    fn mut_lock_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<MvccLock> {
        &mut self.lock
    }

    // repeated .kvrpcpb.MvccWrite writes = 2;

    pub fn clear_writes(&mut self) {
        self.writes.clear();
    }

    // Param is passed by value, moved
    pub fn set_writes(&mut self, v: ::protobuf::RepeatedField<MvccWrite>) {
        self.writes = v;
    }

    // Mutable pointer to the field.
    pub fn mut_writes(&mut self) -> &mut ::protobuf::RepeatedField<MvccWrite> {
        &mut self.writes
    }

    // Take field
    pub fn take_writes(&mut self) -> ::protobuf::RepeatedField<MvccWrite> {
        ::std::mem::replace(&mut self.writes, ::protobuf::RepeatedField::new())
    }

    pub fn get_writes(&self) -> &[MvccWrite] {
        &self.writes
    }

    fn get_writes_for_reflect(&self) -> &::protobuf::RepeatedField<MvccWrite> {
        &self.writes
    }

    fn mut_writes_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<MvccWrite> {
        &mut self.writes
    }

    // repeated .kvrpcpb.MvccValue values = 3;

    pub fn clear_values(&mut self) {
        self.values.clear();
    }

    // Param is passed by value, moved
    pub fn set_values(&mut self, v: ::protobuf::RepeatedField<MvccValue>) {
        self.values = v;
    }

    // Mutable pointer to the field.
    pub fn mut_values(&mut self) -> &mut ::protobuf::RepeatedField<MvccValue> {
        &mut self.values
    }

    // Take field
    pub fn take_values(&mut self) -> ::protobuf::RepeatedField<MvccValue> {
        ::std::mem::replace(&mut self.values, ::protobuf::RepeatedField::new())
    }

    pub fn get_values(&self) -> &[MvccValue] {
        &self.values
    }

    fn get_values_for_reflect(&self) -> &::protobuf::RepeatedField<MvccValue> {
        &self.values
    }

    fn mut_values_for_reflect(&mut self) -> &mut ::protobuf::RepeatedField<MvccValue> {
        &mut self.values
    }
}

impl ::protobuf::Message for MvccInfo {
    fn is_initialized(&self) -> bool {
        for v in &self.lock {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.writes {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.values {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.lock)?;
                },
                2 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.writes)?;
                },
                3 => {
                    ::protobuf::rt::read_repeated_message_into(wire_type, is, &mut self.values)?;
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
        if let Some(ref v) = self.lock.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        for value in &self.writes {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        for value in &self.values {
            let len = value.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        };
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.lock.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        for v in &self.writes {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        };
        for v in &self.values {
            os.write_tag(3, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for MvccInfo {
    fn new() -> MvccInfo {
        MvccInfo::new()
    }

    fn descriptor_static(_: ::std::option::Option<MvccInfo>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<MvccLock>>(
                    "lock",
                    MvccInfo::get_lock_for_reflect,
                    MvccInfo::mut_lock_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<MvccWrite>>(
                    "writes",
                    MvccInfo::get_writes_for_reflect,
                    MvccInfo::mut_writes_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_repeated_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<MvccValue>>(
                    "values",
                    MvccInfo::get_values_for_reflect,
                    MvccInfo::mut_values_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<MvccInfo>(
                    "MvccInfo",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for MvccInfo {
    fn clear(&mut self) {
        self.clear_lock();
        self.clear_writes();
        self.clear_values();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for MvccInfo {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for MvccInfo {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct MvccGetByKeyRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub key: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for MvccGetByKeyRequest {}

impl MvccGetByKeyRequest {
    pub fn new() -> MvccGetByKeyRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static MvccGetByKeyRequest {
        static mut instance: ::protobuf::lazy::Lazy<MvccGetByKeyRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const MvccGetByKeyRequest,
        };
        unsafe {
            instance.get(MvccGetByKeyRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
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
}

impl ::protobuf::Message for MvccGetByKeyRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
        if let Some(ref v) = self.context.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.key);
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
        if !self.key.is_empty() {
            os.write_bytes(2, &self.key)?;
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

impl ::protobuf::MessageStatic for MvccGetByKeyRequest {
    fn new() -> MvccGetByKeyRequest {
        MvccGetByKeyRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<MvccGetByKeyRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    MvccGetByKeyRequest::get_context_for_reflect,
                    MvccGetByKeyRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    MvccGetByKeyRequest::get_key_for_reflect,
                    MvccGetByKeyRequest::mut_key_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<MvccGetByKeyRequest>(
                    "MvccGetByKeyRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for MvccGetByKeyRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_key();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for MvccGetByKeyRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for MvccGetByKeyRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct MvccGetByKeyResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    pub info: ::protobuf::SingularPtrField<MvccInfo>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for MvccGetByKeyResponse {}

impl MvccGetByKeyResponse {
    pub fn new() -> MvccGetByKeyResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static MvccGetByKeyResponse {
        static mut instance: ::protobuf::lazy::Lazy<MvccGetByKeyResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const MvccGetByKeyResponse,
        };
        unsafe {
            instance.get(MvccGetByKeyResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // .kvrpcpb.MvccInfo info = 3;

    pub fn clear_info(&mut self) {
        self.info.clear();
    }

    pub fn has_info(&self) -> bool {
        self.info.is_some()
    }

    // Param is passed by value, moved
    pub fn set_info(&mut self, v: MvccInfo) {
        self.info = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_info(&mut self) -> &mut MvccInfo {
        if self.info.is_none() {
            self.info.set_default();
        }
        self.info.as_mut().unwrap()
    }

    // Take field
    pub fn take_info(&mut self) -> MvccInfo {
        self.info.take().unwrap_or_else(|| MvccInfo::new())
    }

    pub fn get_info(&self) -> &MvccInfo {
        self.info.as_ref().unwrap_or_else(|| MvccInfo::default_instance())
    }

    fn get_info_for_reflect(&self) -> &::protobuf::SingularPtrField<MvccInfo> {
        &self.info
    }

    fn mut_info_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<MvccInfo> {
        &mut self.info
    }
}

impl ::protobuf::Message for MvccGetByKeyResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
                },
                3 => {
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
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
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
        }
        if let Some(ref v) = self.info.as_ref() {
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

impl ::protobuf::MessageStatic for MvccGetByKeyResponse {
    fn new() -> MvccGetByKeyResponse {
        MvccGetByKeyResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<MvccGetByKeyResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    MvccGetByKeyResponse::get_region_error_for_reflect,
                    MvccGetByKeyResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    MvccGetByKeyResponse::get_error_for_reflect,
                    MvccGetByKeyResponse::mut_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<MvccInfo>>(
                    "info",
                    MvccGetByKeyResponse::get_info_for_reflect,
                    MvccGetByKeyResponse::mut_info_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<MvccGetByKeyResponse>(
                    "MvccGetByKeyResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for MvccGetByKeyResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.clear_info();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for MvccGetByKeyResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for MvccGetByKeyResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct MvccGetByStartTsRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub start_ts: u64,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for MvccGetByStartTsRequest {}

impl MvccGetByStartTsRequest {
    pub fn new() -> MvccGetByStartTsRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static MvccGetByStartTsRequest {
        static mut instance: ::protobuf::lazy::Lazy<MvccGetByStartTsRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const MvccGetByStartTsRequest,
        };
        unsafe {
            instance.get(MvccGetByStartTsRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // uint64 start_ts = 2;

    pub fn clear_start_ts(&mut self) {
        self.start_ts = 0;
    }

    // Param is passed by value, moved
    pub fn set_start_ts(&mut self, v: u64) {
        self.start_ts = v;
    }

    pub fn get_start_ts(&self) -> u64 {
        self.start_ts
    }

    fn get_start_ts_for_reflect(&self) -> &u64 {
        &self.start_ts
    }

    fn mut_start_ts_for_reflect(&mut self) -> &mut u64 {
        &mut self.start_ts
    }
}

impl ::protobuf::Message for MvccGetByStartTsRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    if wire_type != ::protobuf::wire_format::WireTypeVarint {
                        return ::std::result::Result::Err(::protobuf::rt::unexpected_wire_type(wire_type));
                    }
                    let tmp = is.read_uint64()?;
                    self.start_ts = tmp;
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
        if self.start_ts != 0 {
            my_size += ::protobuf::rt::value_size(2, self.start_ts, ::protobuf::wire_format::WireTypeVarint);
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
        if self.start_ts != 0 {
            os.write_uint64(2, self.start_ts)?;
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

impl ::protobuf::MessageStatic for MvccGetByStartTsRequest {
    fn new() -> MvccGetByStartTsRequest {
        MvccGetByStartTsRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<MvccGetByStartTsRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    MvccGetByStartTsRequest::get_context_for_reflect,
                    MvccGetByStartTsRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeUint64>(
                    "start_ts",
                    MvccGetByStartTsRequest::get_start_ts_for_reflect,
                    MvccGetByStartTsRequest::mut_start_ts_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<MvccGetByStartTsRequest>(
                    "MvccGetByStartTsRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for MvccGetByStartTsRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_start_ts();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for MvccGetByStartTsRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for MvccGetByStartTsRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct MvccGetByStartTsResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub error: ::std::string::String,
    pub key: ::std::vec::Vec<u8>,
    pub info: ::protobuf::SingularPtrField<MvccInfo>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for MvccGetByStartTsResponse {}

impl MvccGetByStartTsResponse {
    pub fn new() -> MvccGetByStartTsResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static MvccGetByStartTsResponse {
        static mut instance: ::protobuf::lazy::Lazy<MvccGetByStartTsResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const MvccGetByStartTsResponse,
        };
        unsafe {
            instance.get(MvccGetByStartTsResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // string error = 2;

    pub fn clear_error(&mut self) {
        self.error.clear();
    }

    // Param is passed by value, moved
    pub fn set_error(&mut self, v: ::std::string::String) {
        self.error = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_error(&mut self) -> &mut ::std::string::String {
        &mut self.error
    }

    // Take field
    pub fn take_error(&mut self) -> ::std::string::String {
        ::std::mem::replace(&mut self.error, ::std::string::String::new())
    }

    pub fn get_error(&self) -> &str {
        &self.error
    }

    fn get_error_for_reflect(&self) -> &::std::string::String {
        &self.error
    }

    fn mut_error_for_reflect(&mut self) -> &mut ::std::string::String {
        &mut self.error
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

    // .kvrpcpb.MvccInfo info = 4;

    pub fn clear_info(&mut self) {
        self.info.clear();
    }

    pub fn has_info(&self) -> bool {
        self.info.is_some()
    }

    // Param is passed by value, moved
    pub fn set_info(&mut self, v: MvccInfo) {
        self.info = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_info(&mut self) -> &mut MvccInfo {
        if self.info.is_none() {
            self.info.set_default();
        }
        self.info.as_mut().unwrap()
    }

    // Take field
    pub fn take_info(&mut self) -> MvccInfo {
        self.info.take().unwrap_or_else(|| MvccInfo::new())
    }

    pub fn get_info(&self) -> &MvccInfo {
        self.info.as_ref().unwrap_or_else(|| MvccInfo::default_instance())
    }

    fn get_info_for_reflect(&self) -> &::protobuf::SingularPtrField<MvccInfo> {
        &self.info
    }

    fn mut_info_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<MvccInfo> {
        &mut self.info
    }
}

impl ::protobuf::Message for MvccGetByStartTsResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_proto3_string_into(wire_type, is, &mut self.error)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.key)?;
                },
                4 => {
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if !self.error.is_empty() {
            my_size += ::protobuf::rt::string_size(2, &self.error);
        }
        if !self.key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(3, &self.key);
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
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if !self.error.is_empty() {
            os.write_string(2, &self.error)?;
        }
        if !self.key.is_empty() {
            os.write_bytes(3, &self.key)?;
        }
        if let Some(ref v) = self.info.as_ref() {
            os.write_tag(4, ::protobuf::wire_format::WireTypeLengthDelimited)?;
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

impl ::protobuf::MessageStatic for MvccGetByStartTsResponse {
    fn new() -> MvccGetByStartTsResponse {
        MvccGetByStartTsResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<MvccGetByStartTsResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    MvccGetByStartTsResponse::get_region_error_for_reflect,
                    MvccGetByStartTsResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeString>(
                    "error",
                    MvccGetByStartTsResponse::get_error_for_reflect,
                    MvccGetByStartTsResponse::mut_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "key",
                    MvccGetByStartTsResponse::get_key_for_reflect,
                    MvccGetByStartTsResponse::mut_key_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<MvccInfo>>(
                    "info",
                    MvccGetByStartTsResponse::get_info_for_reflect,
                    MvccGetByStartTsResponse::mut_info_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<MvccGetByStartTsResponse>(
                    "MvccGetByStartTsResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for MvccGetByStartTsResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_error();
        self.clear_key();
        self.clear_info();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for MvccGetByStartTsResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for MvccGetByStartTsResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct SplitRegionRequest {
    // message fields
    pub context: ::protobuf::SingularPtrField<Context>,
    pub split_key: ::std::vec::Vec<u8>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for SplitRegionRequest {}

impl SplitRegionRequest {
    pub fn new() -> SplitRegionRequest {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static SplitRegionRequest {
        static mut instance: ::protobuf::lazy::Lazy<SplitRegionRequest> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const SplitRegionRequest,
        };
        unsafe {
            instance.get(SplitRegionRequest::new)
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
    pub fn set_context(&mut self, v: Context) {
        self.context = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_context(&mut self) -> &mut Context {
        if self.context.is_none() {
            self.context.set_default();
        }
        self.context.as_mut().unwrap()
    }

    // Take field
    pub fn take_context(&mut self) -> Context {
        self.context.take().unwrap_or_else(|| Context::new())
    }

    pub fn get_context(&self) -> &Context {
        self.context.as_ref().unwrap_or_else(|| Context::default_instance())
    }

    fn get_context_for_reflect(&self) -> &::protobuf::SingularPtrField<Context> {
        &self.context
    }

    fn mut_context_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<Context> {
        &mut self.context
    }

    // bytes split_key = 2;

    pub fn clear_split_key(&mut self) {
        self.split_key.clear();
    }

    // Param is passed by value, moved
    pub fn set_split_key(&mut self, v: ::std::vec::Vec<u8>) {
        self.split_key = v;
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_split_key(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.split_key
    }

    // Take field
    pub fn take_split_key(&mut self) -> ::std::vec::Vec<u8> {
        ::std::mem::replace(&mut self.split_key, ::std::vec::Vec::new())
    }

    pub fn get_split_key(&self) -> &[u8] {
        &self.split_key
    }

    fn get_split_key_for_reflect(&self) -> &::std::vec::Vec<u8> {
        &self.split_key
    }

    fn mut_split_key_for_reflect(&mut self) -> &mut ::std::vec::Vec<u8> {
        &mut self.split_key
    }
}

impl ::protobuf::Message for SplitRegionRequest {
    fn is_initialized(&self) -> bool {
        for v in &self.context {
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
                    ::protobuf::rt::read_singular_proto3_bytes_into(wire_type, is, &mut self.split_key)?;
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
        if !self.split_key.is_empty() {
            my_size += ::protobuf::rt::bytes_size(2, &self.split_key);
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
        if !self.split_key.is_empty() {
            os.write_bytes(2, &self.split_key)?;
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

impl ::protobuf::MessageStatic for SplitRegionRequest {
    fn new() -> SplitRegionRequest {
        SplitRegionRequest::new()
    }

    fn descriptor_static(_: ::std::option::Option<SplitRegionRequest>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<Context>>(
                    "context",
                    SplitRegionRequest::get_context_for_reflect,
                    SplitRegionRequest::mut_context_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_simple_field_accessor::<_, ::protobuf::types::ProtobufTypeBytes>(
                    "split_key",
                    SplitRegionRequest::get_split_key_for_reflect,
                    SplitRegionRequest::mut_split_key_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<SplitRegionRequest>(
                    "SplitRegionRequest",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for SplitRegionRequest {
    fn clear(&mut self) {
        self.clear_context();
        self.clear_split_key();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for SplitRegionRequest {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for SplitRegionRequest {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(PartialEq,Clone,Default)]
pub struct SplitRegionResponse {
    // message fields
    pub region_error: ::protobuf::SingularPtrField<super::errorpb::Error>,
    pub left: ::protobuf::SingularPtrField<super::metapb::Region>,
    pub right: ::protobuf::SingularPtrField<super::metapb::Region>,
    // special fields
    unknown_fields: ::protobuf::UnknownFields,
    cached_size: ::protobuf::CachedSize,
}

// see codegen.rs for the explanation why impl Sync explicitly
unsafe impl ::std::marker::Sync for SplitRegionResponse {}

impl SplitRegionResponse {
    pub fn new() -> SplitRegionResponse {
        ::std::default::Default::default()
    }

    pub fn default_instance() -> &'static SplitRegionResponse {
        static mut instance: ::protobuf::lazy::Lazy<SplitRegionResponse> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const SplitRegionResponse,
        };
        unsafe {
            instance.get(SplitRegionResponse::new)
        }
    }

    // .errorpb.Error region_error = 1;

    pub fn clear_region_error(&mut self) {
        self.region_error.clear();
    }

    pub fn has_region_error(&self) -> bool {
        self.region_error.is_some()
    }

    // Param is passed by value, moved
    pub fn set_region_error(&mut self, v: super::errorpb::Error) {
        self.region_error = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_region_error(&mut self) -> &mut super::errorpb::Error {
        if self.region_error.is_none() {
            self.region_error.set_default();
        }
        self.region_error.as_mut().unwrap()
    }

    // Take field
    pub fn take_region_error(&mut self) -> super::errorpb::Error {
        self.region_error.take().unwrap_or_else(|| super::errorpb::Error::new())
    }

    pub fn get_region_error(&self) -> &super::errorpb::Error {
        self.region_error.as_ref().unwrap_or_else(|| super::errorpb::Error::default_instance())
    }

    fn get_region_error_for_reflect(&self) -> &::protobuf::SingularPtrField<super::errorpb::Error> {
        &self.region_error
    }

    fn mut_region_error_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::errorpb::Error> {
        &mut self.region_error
    }

    // .metapb.Region left = 2;

    pub fn clear_left(&mut self) {
        self.left.clear();
    }

    pub fn has_left(&self) -> bool {
        self.left.is_some()
    }

    // Param is passed by value, moved
    pub fn set_left(&mut self, v: super::metapb::Region) {
        self.left = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_left(&mut self) -> &mut super::metapb::Region {
        if self.left.is_none() {
            self.left.set_default();
        }
        self.left.as_mut().unwrap()
    }

    // Take field
    pub fn take_left(&mut self) -> super::metapb::Region {
        self.left.take().unwrap_or_else(|| super::metapb::Region::new())
    }

    pub fn get_left(&self) -> &super::metapb::Region {
        self.left.as_ref().unwrap_or_else(|| super::metapb::Region::default_instance())
    }

    fn get_left_for_reflect(&self) -> &::protobuf::SingularPtrField<super::metapb::Region> {
        &self.left
    }

    fn mut_left_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::metapb::Region> {
        &mut self.left
    }

    // .metapb.Region right = 3;

    pub fn clear_right(&mut self) {
        self.right.clear();
    }

    pub fn has_right(&self) -> bool {
        self.right.is_some()
    }

    // Param is passed by value, moved
    pub fn set_right(&mut self, v: super::metapb::Region) {
        self.right = ::protobuf::SingularPtrField::some(v);
    }

    // Mutable pointer to the field.
    // If field is not initialized, it is initialized with default value first.
    pub fn mut_right(&mut self) -> &mut super::metapb::Region {
        if self.right.is_none() {
            self.right.set_default();
        }
        self.right.as_mut().unwrap()
    }

    // Take field
    pub fn take_right(&mut self) -> super::metapb::Region {
        self.right.take().unwrap_or_else(|| super::metapb::Region::new())
    }

    pub fn get_right(&self) -> &super::metapb::Region {
        self.right.as_ref().unwrap_or_else(|| super::metapb::Region::default_instance())
    }

    fn get_right_for_reflect(&self) -> &::protobuf::SingularPtrField<super::metapb::Region> {
        &self.right
    }

    fn mut_right_for_reflect(&mut self) -> &mut ::protobuf::SingularPtrField<super::metapb::Region> {
        &mut self.right
    }
}

impl ::protobuf::Message for SplitRegionResponse {
    fn is_initialized(&self) -> bool {
        for v in &self.region_error {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.left {
            if !v.is_initialized() {
                return false;
            }
        };
        for v in &self.right {
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
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.region_error)?;
                },
                2 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.left)?;
                },
                3 => {
                    ::protobuf::rt::read_singular_message_into(wire_type, is, &mut self.right)?;
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
        if let Some(ref v) = self.region_error.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.left.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        if let Some(ref v) = self.right.as_ref() {
            let len = v.compute_size();
            my_size += 1 + ::protobuf::rt::compute_raw_varint32_size(len) + len;
        }
        my_size += ::protobuf::rt::unknown_fields_size(self.get_unknown_fields());
        self.cached_size.set(my_size);
        my_size
    }

    fn write_to_with_cached_sizes(&self, os: &mut ::protobuf::CodedOutputStream) -> ::protobuf::ProtobufResult<()> {
        if let Some(ref v) = self.region_error.as_ref() {
            os.write_tag(1, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.left.as_ref() {
            os.write_tag(2, ::protobuf::wire_format::WireTypeLengthDelimited)?;
            os.write_raw_varint32(v.get_cached_size())?;
            v.write_to_with_cached_sizes(os)?;
        }
        if let Some(ref v) = self.right.as_ref() {
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

impl ::protobuf::MessageStatic for SplitRegionResponse {
    fn new() -> SplitRegionResponse {
        SplitRegionResponse::new()
    }

    fn descriptor_static(_: ::std::option::Option<SplitRegionResponse>) -> &'static ::protobuf::reflect::MessageDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::MessageDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::MessageDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                let mut fields = ::std::vec::Vec::new();
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::errorpb::Error>>(
                    "region_error",
                    SplitRegionResponse::get_region_error_for_reflect,
                    SplitRegionResponse::mut_region_error_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::metapb::Region>>(
                    "left",
                    SplitRegionResponse::get_left_for_reflect,
                    SplitRegionResponse::mut_left_for_reflect,
                ));
                fields.push(::protobuf::reflect::accessor::make_singular_ptr_field_accessor::<_, ::protobuf::types::ProtobufTypeMessage<super::metapb::Region>>(
                    "right",
                    SplitRegionResponse::get_right_for_reflect,
                    SplitRegionResponse::mut_right_for_reflect,
                ));
                ::protobuf::reflect::MessageDescriptor::new::<SplitRegionResponse>(
                    "SplitRegionResponse",
                    fields,
                    file_descriptor_proto()
                )
            })
        }
    }
}

impl ::protobuf::Clear for SplitRegionResponse {
    fn clear(&mut self) {
        self.clear_region_error();
        self.clear_left();
        self.clear_right();
        self.unknown_fields.clear();
    }
}

impl ::std::fmt::Debug for SplitRegionResponse {
    fn fmt(&self, f: &mut ::std::fmt::Formatter) -> ::std::fmt::Result {
        ::protobuf::text_format::fmt(self, f)
    }
}

impl ::protobuf::reflect::ProtobufValue for SplitRegionResponse {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Message(self)
    }
}

#[derive(Clone,PartialEq,Eq,Debug,Hash)]
pub enum CommandPri {
    Normal = 0,
    Low = 1,
    High = 2,
}

impl ::protobuf::ProtobufEnum for CommandPri {
    fn value(&self) -> i32 {
        *self as i32
    }

    fn from_i32(value: i32) -> ::std::option::Option<CommandPri> {
        match value {
            0 => ::std::option::Option::Some(CommandPri::Normal),
            1 => ::std::option::Option::Some(CommandPri::Low),
            2 => ::std::option::Option::Some(CommandPri::High),
            _ => ::std::option::Option::None
        }
    }

    fn values() -> &'static [Self] {
        static values: &'static [CommandPri] = &[
            CommandPri::Normal,
            CommandPri::Low,
            CommandPri::High,
        ];
        values
    }

    fn enum_descriptor_static(_: ::std::option::Option<CommandPri>) -> &'static ::protobuf::reflect::EnumDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::EnumDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::EnumDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                ::protobuf::reflect::EnumDescriptor::new("CommandPri", file_descriptor_proto())
            })
        }
    }
}

impl ::std::marker::Copy for CommandPri {
}

impl ::std::default::Default for CommandPri {
    fn default() -> Self {
        CommandPri::Normal
    }
}

impl ::protobuf::reflect::ProtobufValue for CommandPri {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Enum(self.descriptor())
    }
}

#[derive(Clone,PartialEq,Eq,Debug,Hash)]
pub enum IsolationLevel {
    SI = 0,
    RC = 1,
}

impl ::protobuf::ProtobufEnum for IsolationLevel {
    fn value(&self) -> i32 {
        *self as i32
    }

    fn from_i32(value: i32) -> ::std::option::Option<IsolationLevel> {
        match value {
            0 => ::std::option::Option::Some(IsolationLevel::SI),
            1 => ::std::option::Option::Some(IsolationLevel::RC),
            _ => ::std::option::Option::None
        }
    }

    fn values() -> &'static [Self] {
        static values: &'static [IsolationLevel] = &[
            IsolationLevel::SI,
            IsolationLevel::RC,
        ];
        values
    }

    fn enum_descriptor_static(_: ::std::option::Option<IsolationLevel>) -> &'static ::protobuf::reflect::EnumDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::EnumDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::EnumDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                ::protobuf::reflect::EnumDescriptor::new("IsolationLevel", file_descriptor_proto())
            })
        }
    }
}

impl ::std::marker::Copy for IsolationLevel {
}

impl ::std::default::Default for IsolationLevel {
    fn default() -> Self {
        IsolationLevel::SI
    }
}

impl ::protobuf::reflect::ProtobufValue for IsolationLevel {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Enum(self.descriptor())
    }
}

#[derive(Clone,PartialEq,Eq,Debug,Hash)]
pub enum Op {
    Put = 0,
    Del = 1,
    Lock = 2,
    Rollback = 3,
}

impl ::protobuf::ProtobufEnum for Op {
    fn value(&self) -> i32 {
        *self as i32
    }

    fn from_i32(value: i32) -> ::std::option::Option<Op> {
        match value {
            0 => ::std::option::Option::Some(Op::Put),
            1 => ::std::option::Option::Some(Op::Del),
            2 => ::std::option::Option::Some(Op::Lock),
            3 => ::std::option::Option::Some(Op::Rollback),
            _ => ::std::option::Option::None
        }
    }

    fn values() -> &'static [Self] {
        static values: &'static [Op] = &[
            Op::Put,
            Op::Del,
            Op::Lock,
            Op::Rollback,
        ];
        values
    }

    fn enum_descriptor_static(_: ::std::option::Option<Op>) -> &'static ::protobuf::reflect::EnumDescriptor {
        static mut descriptor: ::protobuf::lazy::Lazy<::protobuf::reflect::EnumDescriptor> = ::protobuf::lazy::Lazy {
            lock: ::protobuf::lazy::ONCE_INIT,
            ptr: 0 as *const ::protobuf::reflect::EnumDescriptor,
        };
        unsafe {
            descriptor.get(|| {
                ::protobuf::reflect::EnumDescriptor::new("Op", file_descriptor_proto())
            })
        }
    }
}

impl ::std::marker::Copy for Op {
}

impl ::std::default::Default for Op {
    fn default() -> Self {
        Op::Put
    }
}

impl ::protobuf::reflect::ProtobufValue for Op {
    fn as_ref(&self) -> ::protobuf::reflect::ProtobufValueRef {
        ::protobuf::reflect::ProtobufValueRef::Enum(self.descriptor())
    }
}

static file_descriptor_proto_data: &'static [u8] = b"\
    \n\rkvrpcpb.proto\x12\x07kvrpcpb\x1a\x0cmetapb.proto\x1a\rerrorpb.proto\
    \x1a\x14gogoproto/gogo.proto\"}\n\x08LockInfo\x12!\n\x0cprimary_lock\x18\
    \x01\x20\x01(\x0cR\x0bprimaryLock\x12!\n\x0clock_version\x18\x02\x20\x01\
    (\x04R\x0blockVersion\x12\x10\n\x03key\x18\x03\x20\x01(\x0cR\x03key\x12\
    \x19\n\x08lock_ttl\x18\x04\x20\x01(\x04R\x07lockTtl\"\x9d\x01\n\x08KeyEr\
    ror\x12)\n\x06locked\x18\x01\x20\x01(\x0b2\x11.kvrpcpb.LockInfoR\x06lock\
    ed\x12\x1c\n\tretryable\x18\x02\x20\x01(\tR\tretryable\x12\x14\n\x05abor\
    t\x18\x03\x20\x01(\tR\x05abort\x122\n\x08conflict\x18\x04\x20\x01(\x0b2\
    \x16.kvrpcpb.WriteConflictR\x08conflict\"w\n\rWriteConflict\x12\x19\n\
    \x08start_ts\x18\x01\x20\x01(\x04R\x07startTs\x12\x1f\n\x0bconflict_ts\
    \x18\x02\x20\x01(\x04R\nconflictTs\x12\x10\n\x03key\x18\x03\x20\x01(\x0c\
    R\x03key\x12\x18\n\x07primary\x18\x04\x20\x01(\x0cR\x07primary\"\x9d\x03\
    \n\x07Context\x12\x1b\n\tregion_id\x18\x01\x20\x01(\x04R\x08regionId\x12\
    6\n\x0cregion_epoch\x18\x02\x20\x01(\x0b2\x13.metapb.RegionEpochR\x0breg\
    ionEpoch\x12\x20\n\x04peer\x18\x03\x20\x01(\x0b2\x0c.metapb.PeerR\x04pee\
    r\x12\x12\n\x04term\x18\x05\x20\x01(\x04R\x04term\x12/\n\x08priority\x18\
    \x06\x20\x01(\x0e2\x13.kvrpcpb.CommandPriR\x08priority\x12@\n\x0fisolati\
    on_level\x18\x07\x20\x01(\x0e2\x17.kvrpcpb.IsolationLevelR\x0eisolationL\
    evel\x12$\n\x0enot_fill_cache\x18\x08\x20\x01(\x08R\x0cnotFillCache\x12\
    \x19\n\x08sync_log\x18\t\x20\x01(\x08R\x07syncLog\x12\x1f\n\x0bhandle_ti\
    me\x18\n\x20\x01(\x08R\nhandleTime\x12\x1f\n\x0bscan_detail\x18\x0b\x20\
    \x01(\x08R\nscanDetailJ\x04\x08\x04\x10\x05R\x0bread_quorum\"D\n\nHandle\
    Time\x12\x17\n\x07wait_ms\x18\x01\x20\x01(\x03R\x06waitMs\x12\x1d\n\npro\
    cess_ms\x18\x02\x20\x01(\x03R\tprocessMs\">\n\x08ScanInfo\x12\x14\n\x05t\
    otal\x18\x01\x20\x01(\x03R\x05total\x12\x1c\n\tprocessed\x18\x02\x20\x01\
    (\x03R\tprocessed\"\x83\x01\n\nScanDetail\x12'\n\x05write\x18\x01\x20\
    \x01(\x0b2\x11.kvrpcpb.ScanInfoR\x05write\x12%\n\x04lock\x18\x02\x20\x01\
    (\x0b2\x11.kvrpcpb.ScanInfoR\x04lock\x12%\n\x04data\x18\x03\x20\x01(\x0b\
    2\x11.kvrpcpb.ScanInfoR\x04data\"y\n\x0bExecDetails\x124\n\x0bhandle_tim\
    e\x18\x01\x20\x01(\x0b2\x13.kvrpcpb.HandleTimeR\nhandleTime\x124\n\x0bsc\
    an_detail\x18\x02\x20\x01(\x0b2\x13.kvrpcpb.ScanDetailR\nscanDetail\"d\n\
    \nGetRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.Context\
    R\x07context\x12\x10\n\x03key\x18\x02\x20\x01(\x0cR\x03key\x12\x18\n\x07\
    version\x18\x03\x20\x01(\x04R\x07version\"\x7f\n\x0bGetResponse\x121\n\
    \x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\
    \x12'\n\x05error\x18\x02\x20\x01(\x0b2\x11.kvrpcpb.KeyErrorR\x05error\
    \x12\x14\n\x05value\x18\x03\x20\x01(\x0cR\x05value\"\xbb\x01\n\x0bScanRe\
    quest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07co\
    ntext\x12\x1b\n\tstart_key\x18\x02\x20\x01(\x0cR\x08startKey\x12\x14\n\
    \x05limit\x18\x03\x20\x01(\rR\x05limit\x12\x18\n\x07version\x18\x04\x20\
    \x01(\x04R\x07version\x12\x19\n\x08key_only\x18\x05\x20\x01(\x08R\x07key\
    Only\x12\x18\n\x07reverse\x18\x06\x20\x01(\x08R\x07reverse\"Y\n\x06KvPai\
    r\x12'\n\x05error\x18\x01\x20\x01(\x0b2\x11.kvrpcpb.KeyErrorR\x05error\
    \x12\x10\n\x03key\x18\x02\x20\x01(\x0cR\x03key\x12\x14\n\x05value\x18\
    \x03\x20\x01(\x0cR\x05value\"h\n\x0cScanResponse\x121\n\x0cregion_error\
    \x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12%\n\x05pairs\
    \x18\x02\x20\x03(\x0b2\x0f.kvrpcpb.KvPairR\x05pairs\"O\n\x08Mutation\x12\
    \x1b\n\x02op\x18\x01\x20\x01(\x0e2\x0b.kvrpcpb.OpR\x02op\x12\x10\n\x03ke\
    y\x18\x02\x20\x01(\x0cR\x03key\x12\x14\n\x05value\x18\x03\x20\x01(\x0cR\
    \x05value\"\x85\x02\n\x0fPrewriteRequest\x12*\n\x07context\x18\x01\x20\
    \x01(\x0b2\x10.kvrpcpb.ContextR\x07context\x12/\n\tmutations\x18\x02\x20\
    \x03(\x0b2\x11.kvrpcpb.MutationR\tmutations\x12!\n\x0cprimary_lock\x18\
    \x03\x20\x01(\x0cR\x0bprimaryLock\x12#\n\rstart_version\x18\x04\x20\x01(\
    \x04R\x0cstartVersion\x12\x19\n\x08lock_ttl\x18\x05\x20\x01(\x04R\x07loc\
    kTtl\x122\n\x15skip_constraint_check\x18\x06\x20\x01(\x08R\x13skipConstr\
    aintCheck\"p\n\x10PrewriteResponse\x121\n\x0cregion_error\x18\x01\x20\
    \x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12)\n\x06errors\x18\x02\
    \x20\x03(\x0b2\x11.kvrpcpb.KeyErrorR\x06errors\"\xa9\x01\n\rCommitReques\
    t\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07contex\
    t\x12#\n\rstart_version\x18\x02\x20\x01(\x04R\x0cstartVersion\x12\x12\n\
    \x04keys\x18\x03\x20\x03(\x0cR\x04keys\x12%\n\x0ecommit_version\x18\x04\
    \x20\x01(\x04R\rcommitVersionJ\x04\x08\x05\x10\x06R\x06binlog\"l\n\x0eCo\
    mmitResponse\x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.Er\
    rorR\x0bregionError\x12'\n\x05error\x18\x02\x20\x01(\x0b2\x11.kvrpcpb.Ke\
    yErrorR\x05error\"g\n\rImportRequest\x12/\n\tmutations\x18\x01\x20\x03(\
    \x0b2\x11.kvrpcpb.MutationR\tmutations\x12%\n\x0ecommit_version\x18\x02\
    \x20\x01(\x04R\rcommitVersion\"Y\n\x0eImportResponse\x121\n\x0cregion_er\
    ror\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12\x14\n\
    \x05error\x18\x02\x20\x01(\tR\x05error\"{\n\x14BatchRollbackRequest\x12*\
    \n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07context\x12#\
    \n\rstart_version\x18\x02\x20\x01(\x04R\x0cstartVersion\x12\x12\n\x04key\
    s\x18\x03\x20\x03(\x0cR\x04keys\"s\n\x15BatchRollbackResponse\x121\n\x0c\
    region_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12\
    '\n\x05error\x18\x02\x20\x01(\x0b2\x11.kvrpcpb.KeyErrorR\x05error\"s\n\
    \x0eCleanupRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.C\
    ontextR\x07context\x12\x10\n\x03key\x18\x02\x20\x01(\x0cR\x03key\x12#\n\
    \rstart_version\x18\x03\x20\x01(\x04R\x0cstartVersion\"\x94\x01\n\x0fCle\
    anupResponse\x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.Er\
    rorR\x0bregionError\x12'\n\x05error\x18\x02\x20\x01(\x0b2\x11.kvrpcpb.Ke\
    yErrorR\x05error\x12%\n\x0ecommit_version\x18\x03\x20\x01(\x04R\rcommitV\
    ersion\"k\n\x0fBatchGetRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\
    \x10.kvrpcpb.ContextR\x07context\x12\x12\n\x04keys\x18\x02\x20\x03(\x0cR\
    \x04keys\x12\x18\n\x07version\x18\x03\x20\x01(\x04R\x07version\"l\n\x10B\
    atchGetResponse\x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb\
    .ErrorR\x0bregionError\x12%\n\x05pairs\x18\x02\x20\x03(\x0b2\x0f.kvrpcpb\
    .KvPairR\x05pairs\"\x91\x01\n\x0fScanLockRequest\x12*\n\x07context\x18\
    \x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07context\x12\x1f\n\x0bmax_vers\
    ion\x18\x02\x20\x01(\x04R\nmaxVersion\x12\x1b\n\tstart_key\x18\x03\x20\
    \x01(\x0cR\x08startKey\x12\x14\n\x05limit\x18\x04\x20\x01(\rR\x05limit\"\
    \x97\x01\n\x10ScanLockResponse\x121\n\x0cregion_error\x18\x01\x20\x01(\
    \x0b2\x0e.errorpb.ErrorR\x0bregionError\x12'\n\x05error\x18\x02\x20\x01(\
    \x0b2\x11.kvrpcpb.KeyErrorR\x05error\x12'\n\x05locks\x18\x03\x20\x03(\
    \x0b2\x11.kvrpcpb.LockInfoR\x05locks\"3\n\x07TxnInfo\x12\x10\n\x03txn\
    \x18\x01\x20\x01(\x04R\x03txn\x12\x16\n\x06status\x18\x02\x20\x01(\x04R\
    \x06status\"\xbb\x01\n\x12ResolveLockRequest\x12*\n\x07context\x18\x01\
    \x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07context\x12#\n\rstart_version\x18\
    \x02\x20\x01(\x04R\x0cstartVersion\x12%\n\x0ecommit_version\x18\x03\x20\
    \x01(\x04R\rcommitVersion\x12-\n\ttxn_infos\x18\x04\x20\x03(\x0b2\x10.kv\
    rpcpb.TxnInfoR\x08txnInfos\"q\n\x13ResolveLockResponse\x121\n\x0cregion_\
    error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12'\n\x05\
    error\x18\x02\x20\x01(\x0b2\x11.kvrpcpb.KeyErrorR\x05error\"V\n\tGCReque\
    st\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07conte\
    xt\x12\x1d\n\nsafe_point\x18\x02\x20\x01(\x04R\tsafePoint\"h\n\nGCRespon\
    se\x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bre\
    gionError\x12'\n\x05error\x18\x02\x20\x01(\x0b2\x11.kvrpcpb.KeyErrorR\
    \x05error\"]\n\rRawGetRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\
    \x10.kvrpcpb.ContextR\x07context\x12\x10\n\x03key\x18\x02\x20\x01(\x0cR\
    \x03key\x12\x0e\n\x02cf\x18\x03\x20\x01(\tR\x02cf\"o\n\x0eRawGetResponse\
    \x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregi\
    onError\x12\x14\n\x05error\x18\x02\x20\x01(\tR\x05error\x12\x14\n\x05val\
    ue\x18\x03\x20\x01(\x0cR\x05value\"s\n\rRawPutRequest\x12*\n\x07context\
    \x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07context\x12\x10\n\x03key\
    \x18\x02\x20\x01(\x0cR\x03key\x12\x14\n\x05value\x18\x03\x20\x01(\x0cR\
    \x05value\x12\x0e\n\x02cf\x18\x04\x20\x01(\tR\x02cf\"Y\n\x0eRawPutRespon\
    se\x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bre\
    gionError\x12\x14\n\x05error\x18\x02\x20\x01(\tR\x05error\"w\n\x12RawBat\
    chPutRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.Context\
    R\x07context\x12%\n\x05pairs\x18\x02\x20\x03(\x0b2\x0f.kvrpcpb.KvPairR\
    \x05pairs\x12\x0e\n\x02cf\x18\x03\x20\x01(\tR\x02cf\"^\n\x13RawBatchPutR\
    esponse\x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\
    \x0bregionError\x12\x14\n\x05error\x18\x02\x20\x01(\tR\x05error\"d\n\x12\
    RawBatchGetRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.C\
    ontextR\x07context\x12\x12\n\x04keys\x18\x02\x20\x03(\x0cR\x04keys\x12\
    \x0e\n\x02cf\x18\x03\x20\x01(\tR\x02cf\"o\n\x13RawBatchGetResponse\x121\
    \n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionErr\
    or\x12%\n\x05pairs\x18\x02\x20\x03(\x0b2\x0f.kvrpcpb.KvPairR\x05pairs\"`\
    \n\x10RawDeleteRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpc\
    pb.ContextR\x07context\x12\x10\n\x03key\x18\x02\x20\x01(\x0cR\x03key\x12\
    \x0e\n\x02cf\x18\x03\x20\x01(\tR\x02cf\"\\\n\x11RawDeleteResponse\x121\n\
    \x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\
    \x12\x14\n\x05error\x18\x02\x20\x01(\tR\x05error\"g\n\x15RawBatchDeleteR\
    equest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07c\
    ontext\x12\x12\n\x04keys\x18\x02\x20\x03(\x0cR\x04keys\x12\x0e\n\x02cf\
    \x18\x03\x20\x01(\tR\x02cf\"a\n\x16RawBatchDeleteResponse\x121\n\x0cregi\
    on_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12\x14\
    \n\x05error\x18\x02\x20\x01(\tR\x05error\"v\n\x12DeleteRangeRequest\x12*\
    \n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07context\x12\
    \x1b\n\tstart_key\x18\x02\x20\x01(\x0cR\x08startKey\x12\x17\n\x07end_key\
    \x18\x03\x20\x01(\x0cR\x06endKey\"^\n\x13DeleteRangeResponse\x121\n\x0cr\
    egion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12\
    \x14\n\x05error\x18\x02\x20\x01(\tR\x05error\"\x89\x01\n\x15RawDeleteRan\
    geRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\
    \x07context\x12\x1b\n\tstart_key\x18\x02\x20\x01(\x0cR\x08startKey\x12\
    \x17\n\x07end_key\x18\x03\x20\x01(\x0cR\x06endKey\x12\x0e\n\x02cf\x18\
    \x04\x20\x01(\tR\x02cf\"a\n\x16RawDeleteRangeResponse\x121\n\x0cregion_e\
    rror\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12\x14\n\
    \x05error\x18\x02\x20\x01(\tR\x05error\"\x9a\x01\n\x0eRawScanRequest\x12\
    *\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07context\x12\
    \x1b\n\tstart_key\x18\x02\x20\x01(\x0cR\x08startKey\x12\x14\n\x05limit\
    \x18\x03\x20\x01(\rR\x05limit\x12\x19\n\x08key_only\x18\x04\x20\x01(\x08\
    R\x07keyOnly\x12\x0e\n\x02cf\x18\x05\x20\x01(\tR\x02cf\"g\n\x0fRawScanRe\
    sponse\x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\
    \x0bregionError\x12!\n\x03kvs\x18\x02\x20\x03(\x0b2\x0f.kvrpcpb.KvPairR\
    \x03kvs\"@\n\x08KeyRange\x12\x1b\n\tstart_key\x18\x01\x20\x01(\x0cR\x08s\
    tartKey\x12\x17\n\x07end_key\x18\x02\x20\x01(\x0cR\x06endKey\"\xb6\x01\n\
    \x13RawBatchScanRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrp\
    cpb.ContextR\x07context\x12)\n\x06ranges\x18\x02\x20\x03(\x0b2\x11.kvrpc\
    pb.KeyRangeR\x06ranges\x12\x1d\n\neach_limit\x18\x03\x20\x01(\rR\teachLi\
    mit\x12\x19\n\x08key_only\x18\x04\x20\x01(\x08R\x07keyOnly\x12\x0e\n\x02\
    cf\x18\x05\x20\x01(\tR\x02cf\"l\n\x14RawBatchScanResponse\x121\n\x0cregi\
    on_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12!\n\
    \x03kvs\x18\x02\x20\x03(\x0b2\x0f.kvrpcpb.KvPairR\x03kvs\"\x85\x01\n\tMv\
    ccWrite\x12\x1f\n\x04type\x18\x01\x20\x01(\x0e2\x0b.kvrpcpb.OpR\x04type\
    \x12\x19\n\x08start_ts\x18\x02\x20\x01(\x04R\x07startTs\x12\x1b\n\tcommi\
    t_ts\x18\x03\x20\x01(\x04R\x08commitTs\x12\x1f\n\x0bshort_value\x18\x04\
    \x20\x01(\x0cR\nshortValue\"<\n\tMvccValue\x12\x19\n\x08start_ts\x18\x01\
    \x20\x01(\x04R\x07startTs\x12\x14\n\x05value\x18\x02\x20\x01(\x0cR\x05va\
    lue\"\x81\x01\n\x08MvccLock\x12\x1f\n\x04type\x18\x01\x20\x01(\x0e2\x0b.\
    kvrpcpb.OpR\x04type\x12\x19\n\x08start_ts\x18\x02\x20\x01(\x04R\x07start\
    Ts\x12\x18\n\x07primary\x18\x03\x20\x01(\x0cR\x07primary\x12\x1f\n\x0bsh\
    ort_value\x18\x04\x20\x01(\x0cR\nshortValue\"\x89\x01\n\x08MvccInfo\x12%\
    \n\x04lock\x18\x01\x20\x01(\x0b2\x11.kvrpcpb.MvccLockR\x04lock\x12*\n\
    \x06writes\x18\x02\x20\x03(\x0b2\x12.kvrpcpb.MvccWriteR\x06writes\x12*\n\
    \x06values\x18\x03\x20\x03(\x0b2\x12.kvrpcpb.MvccValueR\x06values\"S\n\
    \x13MvccGetByKeyRequest\x12*\n\x07context\x18\x01\x20\x01(\x0b2\x10.kvrp\
    cpb.ContextR\x07context\x12\x10\n\x03key\x18\x02\x20\x01(\x0cR\x03key\"\
    \x86\x01\n\x14MvccGetByKeyResponse\x121\n\x0cregion_error\x18\x01\x20\
    \x01(\x0b2\x0e.errorpb.ErrorR\x0bregionError\x12\x14\n\x05error\x18\x02\
    \x20\x01(\tR\x05error\x12%\n\x04info\x18\x03\x20\x01(\x0b2\x11.kvrpcpb.M\
    vccInfoR\x04info\"`\n\x17MvccGetByStartTsRequest\x12*\n\x07context\x18\
    \x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07context\x12\x19\n\x08start_ts\
    \x18\x02\x20\x01(\x04R\x07startTs\"\x9c\x01\n\x18MvccGetByStartTsRespons\
    e\x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0breg\
    ionError\x12\x14\n\x05error\x18\x02\x20\x01(\tR\x05error\x12\x10\n\x03ke\
    y\x18\x03\x20\x01(\x0cR\x03key\x12%\n\x04info\x18\x04\x20\x01(\x0b2\x11.\
    kvrpcpb.MvccInfoR\x04info\"]\n\x12SplitRegionRequest\x12*\n\x07context\
    \x18\x01\x20\x01(\x0b2\x10.kvrpcpb.ContextR\x07context\x12\x1b\n\tsplit_\
    key\x18\x02\x20\x01(\x0cR\x08splitKey\"\x92\x01\n\x13SplitRegionResponse\
    \x121\n\x0cregion_error\x18\x01\x20\x01(\x0b2\x0e.errorpb.ErrorR\x0bregi\
    onError\x12\"\n\x04left\x18\x02\x20\x01(\x0b2\x0e.metapb.RegionR\x04left\
    \x12$\n\x05right\x18\x03\x20\x01(\x0b2\x0e.metapb.RegionR\x05right*+\n\n\
    CommandPri\x12\n\n\x06Normal\x10\0\x12\x07\n\x03Low\x10\x01\x12\x08\n\
    \x04High\x10\x02*\x20\n\x0eIsolationLevel\x12\x06\n\x02SI\x10\0\x12\x06\
    \n\x02RC\x10\x01*.\n\x02Op\x12\x07\n\x03Put\x10\0\x12\x07\n\x03Del\x10\
    \x01\x12\x08\n\x04Lock\x10\x02\x12\x0c\n\x08Rollback\x10\x03B&\n\x18com.\
    pingcap.tikv.kvproto\xe0\xe2\x1e\x01\xc8\xe2\x1e\x01\xd0\xe2\x1e\x01J\
    \xf5\x89\x01\n\x07\x12\x05\0\0\xa3\x03\x01\n\x08\n\x01\x0c\x12\x03\0\0\
    \x12\n\x08\n\x01\x02\x12\x03\x01\x08\x0f\n\t\n\x02\x03\0\x12\x03\x03\x07\
    \x15\n\t\n\x02\x03\x01\x12\x03\x04\x07\x16\n\t\n\x02\x03\x02\x12\x03\x05\
    \x07\x1d\n\x08\n\x01\x08\x12\x03\x07\0(\n\x0b\n\x04\x08\xe7\x07\0\x12\
    \x03\x07\0(\n\x0c\n\x05\x08\xe7\x07\0\x02\x12\x03\x07\x07\x20\n\r\n\x06\
    \x08\xe7\x07\0\x02\0\x12\x03\x07\x07\x20\n\x0e\n\x07\x08\xe7\x07\0\x02\0\
    \x01\x12\x03\x07\x08\x1f\n\x0c\n\x05\x08\xe7\x07\0\x03\x12\x03\x07#'\n\
    \x08\n\x01\x08\x12\x03\x08\0$\n\x0b\n\x04\x08\xe7\x07\x01\x12\x03\x08\0$\
    \n\x0c\n\x05\x08\xe7\x07\x01\x02\x12\x03\x08\x07\x1c\n\r\n\x06\x08\xe7\
    \x07\x01\x02\0\x12\x03\x08\x07\x1c\n\x0e\n\x07\x08\xe7\x07\x01\x02\0\x01\
    \x12\x03\x08\x08\x1b\n\x0c\n\x05\x08\xe7\x07\x01\x03\x12\x03\x08\x1f#\n\
    \x08\n\x01\x08\x12\x03\t\0*\n\x0b\n\x04\x08\xe7\x07\x02\x12\x03\t\0*\n\
    \x0c\n\x05\x08\xe7\x07\x02\x02\x12\x03\t\x07\"\n\r\n\x06\x08\xe7\x07\x02\
    \x02\0\x12\x03\t\x07\"\n\x0e\n\x07\x08\xe7\x07\x02\x02\0\x01\x12\x03\t\
    \x08!\n\x0c\n\x05\x08\xe7\x07\x02\x03\x12\x03\t%)\n\x08\n\x01\x08\x12\
    \x03\x0b\01\n\x0b\n\x04\x08\xe7\x07\x03\x12\x03\x0b\01\n\x0c\n\x05\x08\
    \xe7\x07\x03\x02\x12\x03\x0b\x07\x13\n\r\n\x06\x08\xe7\x07\x03\x02\0\x12\
    \x03\x0b\x07\x13\n\x0e\n\x07\x08\xe7\x07\x03\x02\0\x01\x12\x03\x0b\x07\
    \x13\n\x0c\n\x05\x08\xe7\x07\x03\x07\x12\x03\x0b\x160\n\n\n\x02\x04\0\
    \x12\x04\r\0\x12\x01\n\n\n\x03\x04\0\x01\x12\x03\r\x08\x10\n\x0b\n\x04\
    \x04\0\x02\0\x12\x03\x0e\x04\x1b\n\r\n\x05\x04\0\x02\0\x04\x12\x04\x0e\
    \x04\r\x12\n\x0c\n\x05\x04\0\x02\0\x05\x12\x03\x0e\x04\t\n\x0c\n\x05\x04\
    \0\x02\0\x01\x12\x03\x0e\n\x16\n\x0c\n\x05\x04\0\x02\0\x03\x12\x03\x0e\
    \x19\x1a\n\x0b\n\x04\x04\0\x02\x01\x12\x03\x0f\x04\x1c\n\r\n\x05\x04\0\
    \x02\x01\x04\x12\x04\x0f\x04\x0e\x1b\n\x0c\n\x05\x04\0\x02\x01\x05\x12\
    \x03\x0f\x04\n\n\x0c\n\x05\x04\0\x02\x01\x01\x12\x03\x0f\x0b\x17\n\x0c\n\
    \x05\x04\0\x02\x01\x03\x12\x03\x0f\x1a\x1b\n\x0b\n\x04\x04\0\x02\x02\x12\
    \x03\x10\x04\x12\n\r\n\x05\x04\0\x02\x02\x04\x12\x04\x10\x04\x0f\x1c\n\
    \x0c\n\x05\x04\0\x02\x02\x05\x12\x03\x10\x04\t\n\x0c\n\x05\x04\0\x02\x02\
    \x01\x12\x03\x10\n\r\n\x0c\n\x05\x04\0\x02\x02\x03\x12\x03\x10\x10\x11\n\
    \x0b\n\x04\x04\0\x02\x03\x12\x03\x11\x04\x18\n\r\n\x05\x04\0\x02\x03\x04\
    \x12\x04\x11\x04\x10\x12\n\x0c\n\x05\x04\0\x02\x03\x05\x12\x03\x11\x04\n\
    \n\x0c\n\x05\x04\0\x02\x03\x01\x12\x03\x11\x0b\x13\n\x0c\n\x05\x04\0\x02\
    \x03\x03\x12\x03\x11\x16\x17\n\n\n\x02\x04\x01\x12\x04\x14\0\x19\x01\n\n\
    \n\x03\x04\x01\x01\x12\x03\x14\x08\x10\nD\n\x04\x04\x01\x02\0\x12\x03\
    \x15\x04\x18\"7\x20Client\x20should\x20backoff\x20or\x20cleanup\x20the\
    \x20lock\x20then\x20retry.\n\n\r\n\x05\x04\x01\x02\0\x04\x12\x04\x15\x04\
    \x14\x12\n\x0c\n\x05\x04\x01\x02\0\x06\x12\x03\x15\x04\x0c\n\x0c\n\x05\
    \x04\x01\x02\0\x01\x12\x03\x15\r\x13\n\x0c\n\x05\x04\x01\x02\0\x03\x12\
    \x03\x15\x16\x17\n>\n\x04\x04\x01\x02\x01\x12\x03\x16\x04\x19\"1\x20Clie\
    nt\x20may\x20restart\x20the\x20txn.\x20e.g\x20write\x20conflict.\n\n\r\n\
    \x05\x04\x01\x02\x01\x04\x12\x04\x16\x04\x15\x18\n\x0c\n\x05\x04\x01\x02\
    \x01\x05\x12\x03\x16\x04\n\n\x0c\n\x05\x04\x01\x02\x01\x01\x12\x03\x16\
    \x0b\x14\n\x0c\n\x05\x04\x01\x02\x01\x03\x12\x03\x16\x17\x18\n+\n\x04\
    \x04\x01\x02\x02\x12\x03\x17\x04\x15\"\x1e\x20Client\x20should\x20abort\
    \x20the\x20txn.\n\n\r\n\x05\x04\x01\x02\x02\x04\x12\x04\x17\x04\x16\x19\
    \n\x0c\n\x05\x04\x01\x02\x02\x05\x12\x03\x17\x04\n\n\x0c\n\x05\x04\x01\
    \x02\x02\x01\x12\x03\x17\x0b\x10\n\x0c\n\x05\x04\x01\x02\x02\x03\x12\x03\
    \x17\x13\x14\n>\n\x04\x04\x01\x02\x03\x12\x03\x18\x04\x1f\"1\x20Write\
    \x20conflict\x20is\x20moved\x20from\x20retryable\x20to\x20here.\n\n\r\n\
    \x05\x04\x01\x02\x03\x04\x12\x04\x18\x04\x17\x15\n\x0c\n\x05\x04\x01\x02\
    \x03\x06\x12\x03\x18\x04\x11\n\x0c\n\x05\x04\x01\x02\x03\x01\x12\x03\x18\
    \x12\x1a\n\x0c\n\x05\x04\x01\x02\x03\x03\x12\x03\x18\x1d\x1e\n\n\n\x02\
    \x04\x02\x12\x04\x1b\0\x20\x01\n\n\n\x03\x04\x02\x01\x12\x03\x1b\x08\x15\
    \n\x0b\n\x04\x04\x02\x02\0\x12\x03\x1c\x04\x18\n\r\n\x05\x04\x02\x02\0\
    \x04\x12\x04\x1c\x04\x1b\x17\n\x0c\n\x05\x04\x02\x02\0\x05\x12\x03\x1c\
    \x04\n\n\x0c\n\x05\x04\x02\x02\0\x01\x12\x03\x1c\x0b\x13\n\x0c\n\x05\x04\
    \x02\x02\0\x03\x12\x03\x1c\x16\x17\n\x0b\n\x04\x04\x02\x02\x01\x12\x03\
    \x1d\x04\x1b\n\r\n\x05\x04\x02\x02\x01\x04\x12\x04\x1d\x04\x1c\x18\n\x0c\
    \n\x05\x04\x02\x02\x01\x05\x12\x03\x1d\x04\n\n\x0c\n\x05\x04\x02\x02\x01\
    \x01\x12\x03\x1d\x0b\x16\n\x0c\n\x05\x04\x02\x02\x01\x03\x12\x03\x1d\x19\
    \x1a\n\x0b\n\x04\x04\x02\x02\x02\x12\x03\x1e\x04\x12\n\r\n\x05\x04\x02\
    \x02\x02\x04\x12\x04\x1e\x04\x1d\x1b\n\x0c\n\x05\x04\x02\x02\x02\x05\x12\
    \x03\x1e\x04\t\n\x0c\n\x05\x04\x02\x02\x02\x01\x12\x03\x1e\n\r\n\x0c\n\
    \x05\x04\x02\x02\x02\x03\x12\x03\x1e\x10\x11\n\x0b\n\x04\x04\x02\x02\x03\
    \x12\x03\x1f\x04\x16\n\r\n\x05\x04\x02\x02\x03\x04\x12\x04\x1f\x04\x1e\
    \x12\n\x0c\n\x05\x04\x02\x02\x03\x05\x12\x03\x1f\x04\t\n\x0c\n\x05\x04\
    \x02\x02\x03\x01\x12\x03\x1f\n\x11\n\x0c\n\x05\x04\x02\x02\x03\x03\x12\
    \x03\x1f\x14\x15\n\n\n\x02\x05\0\x12\x04\"\0&\x01\n\n\n\x03\x05\0\x01\
    \x12\x03\"\x05\x0f\n,\n\x04\x05\0\x02\0\x12\x03#\x04\x0f\"\x1f\x20Normal\
    \x20must\x20the\x20default\x20value\n\n\x0c\n\x05\x05\0\x02\0\x01\x12\
    \x03#\x04\n\n\x0c\n\x05\x05\0\x02\0\x02\x12\x03#\r\x0e\n\x0b\n\x04\x05\0\
    \x02\x01\x12\x03$\x04\x0c\n\x0c\n\x05\x05\0\x02\x01\x01\x12\x03$\x04\x07\
    \n\x0c\n\x05\x05\0\x02\x01\x02\x12\x03$\n\x0b\n\x0b\n\x04\x05\0\x02\x02\
    \x12\x03%\x04\r\n\x0c\n\x05\x05\0\x02\x02\x01\x12\x03%\x04\x08\n\x0c\n\
    \x05\x05\0\x02\x02\x02\x12\x03%\x0b\x0c\n\n\n\x02\x05\x01\x12\x04(\0+\
    \x01\n\n\n\x03\x05\x01\x01\x12\x03(\x05\x13\n&\n\x04\x05\x01\x02\0\x12\
    \x03)\x04\x0b\"\x19\x20SI\x20=\x20snapshot\x20isolation\n\n\x0c\n\x05\
    \x05\x01\x02\0\x01\x12\x03)\x04\x06\n\x0c\n\x05\x05\x01\x02\0\x02\x12\
    \x03)\t\n\n\"\n\x04\x05\x01\x02\x01\x12\x03*\x04\x0b\"\x15\x20RC\x20=\
    \x20read\x20committed\n\n\x0c\n\x05\x05\x01\x02\x01\x01\x12\x03*\x04\x06\
    \n\x0c\n\x05\x05\x01\x02\x01\x02\x12\x03*\t\n\n\n\n\x02\x04\x03\x12\x04-\
    \0:\x01\n\n\n\x03\x04\x03\x01\x12\x03-\x08\x0f\n\n\n\x03\x04\x03\t\x12\
    \x03.\r\x0f\n\x0b\n\x04\x04\x03\t\0\x12\x03.\r\x0e\n\x0c\n\x05\x04\x03\t\
    \0\x01\x12\x03.\r\x0e\n\x0c\n\x05\x04\x03\t\0\x02\x12\x03.\r\x0e\n\n\n\
    \x03\x04\x03\n\x12\x03/\r\x1b\n\x0b\n\x04\x04\x03\n\0\x12\x03/\r\x1a\n\
    \x0b\n\x04\x04\x03\x02\0\x12\x030\x04\x19\n\r\n\x05\x04\x03\x02\0\x04\
    \x12\x040\x04/\x1b\n\x0c\n\x05\x04\x03\x02\0\x05\x12\x030\x04\n\n\x0c\n\
    \x05\x04\x03\x02\0\x01\x12\x030\x0b\x14\n\x0c\n\x05\x04\x03\x02\0\x03\
    \x12\x030\x17\x18\n\x0b\n\x04\x04\x03\x02\x01\x12\x031\x04(\n\r\n\x05\
    \x04\x03\x02\x01\x04\x12\x041\x040\x19\n\x0c\n\x05\x04\x03\x02\x01\x06\
    \x12\x031\x04\x16\n\x0c\n\x05\x04\x03\x02\x01\x01\x12\x031\x17#\n\x0c\n\
    \x05\x04\x03\x02\x01\x03\x12\x031&'\n\x0b\n\x04\x04\x03\x02\x02\x12\x032\
    \x04\x19\n\r\n\x05\x04\x03\x02\x02\x04\x12\x042\x041(\n\x0c\n\x05\x04\
    \x03\x02\x02\x06\x12\x032\x04\x0f\n\x0c\n\x05\x04\x03\x02\x02\x01\x12\
    \x032\x10\x14\n\x0c\n\x05\x04\x03\x02\x02\x03\x12\x032\x17\x18\n\x0b\n\
    \x04\x04\x03\x02\x03\x12\x033\x04\x14\n\r\n\x05\x04\x03\x02\x03\x04\x12\
    \x043\x042\x19\n\x0c\n\x05\x04\x03\x02\x03\x05\x12\x033\x04\n\n\x0c\n\
    \x05\x04\x03\x02\x03\x01\x12\x033\x0b\x0f\n\x0c\n\x05\x04\x03\x02\x03\
    \x03\x12\x033\x12\x13\n\x0b\n\x04\x04\x03\x02\x04\x12\x034\x04\x1c\n\r\n\
    \x05\x04\x03\x02\x04\x04\x12\x044\x043\x14\n\x0c\n\x05\x04\x03\x02\x04\
    \x06\x12\x034\x04\x0e\n\x0c\n\x05\x04\x03\x02\x04\x01\x12\x034\x0f\x17\n\
    \x0c\n\x05\x04\x03\x02\x04\x03\x12\x034\x1a\x1b\n\x0b\n\x04\x04\x03\x02\
    \x05\x12\x035\x04'\n\r\n\x05\x04\x03\x02\x05\x04\x12\x045\x044\x1c\n\x0c\
    \n\x05\x04\x03\x02\x05\x06\x12\x035\x04\x12\n\x0c\n\x05\x04\x03\x02\x05\
    \x01\x12\x035\x13\"\n\x0c\n\x05\x04\x03\x02\x05\x03\x12\x035%&\n\x0b\n\
    \x04\x04\x03\x02\x06\x12\x036\x04\x1c\n\r\n\x05\x04\x03\x02\x06\x04\x12\
    \x046\x045'\n\x0c\n\x05\x04\x03\x02\x06\x05\x12\x036\x04\x08\n\x0c\n\x05\
    \x04\x03\x02\x06\x01\x12\x036\t\x17\n\x0c\n\x05\x04\x03\x02\x06\x03\x12\
    \x036\x1a\x1b\n\x0b\n\x04\x04\x03\x02\x07\x12\x037\x04\x16\n\r\n\x05\x04\
    \x03\x02\x07\x04\x12\x047\x046\x1c\n\x0c\n\x05\x04\x03\x02\x07\x05\x12\
    \x037\x04\x08\n\x0c\n\x05\x04\x03\x02\x07\x01\x12\x037\t\x11\n\x0c\n\x05\
    \x04\x03\x02\x07\x03\x12\x037\x14\x15\n3\n\x04\x04\x03\x02\x08\x12\x038\
    \x04\x1a\"&\x20true\x20means\x20return\x20handle\x20time\x20detail\n\n\r\
    \n\x05\x04\x03\x02\x08\x04\x12\x048\x047\x16\n\x0c\n\x05\x04\x03\x02\x08\
    \x05\x12\x038\x04\x08\n\x0c\n\x05\x04\x03\x02\x08\x01\x12\x038\t\x14\n\
    \x0c\n\x05\x04\x03\x02\x08\x03\x12\x038\x17\x19\n1\n\x04\x04\x03\x02\t\
    \x12\x039\x04\x1a\"$\x20true\x20means\x20return\x20scan\x20cf's\x20detai\
    l\n\n\r\n\x05\x04\x03\x02\t\x04\x12\x049\x048\x1a\n\x0c\n\x05\x04\x03\
    \x02\t\x05\x12\x039\x04\x08\n\x0c\n\x05\x04\x03\x02\t\x01\x12\x039\t\x14\
    \n\x0c\n\x05\x04\x03\x02\t\x03\x12\x039\x17\x19\n\n\n\x02\x04\x04\x12\
    \x04<\0?\x01\n\n\n\x03\x04\x04\x01\x12\x03<\x08\x12\n\x1c\n\x04\x04\x04\
    \x02\0\x12\x03=\x04\x16\"\x0f\x20time\x20in\x20queue\n\n\r\n\x05\x04\x04\
    \x02\0\x04\x12\x04=\x04<\x14\n\x0c\n\x05\x04\x04\x02\0\x05\x12\x03=\x04\
    \t\n\x0c\n\x05\x04\x04\x02\0\x01\x12\x03=\n\x11\n\x0c\n\x05\x04\x04\x02\
    \0\x03\x12\x03=\x14\x15\n.\n\x04\x04\x04\x02\x01\x12\x03>\x04\x19\"!\x20\
    process\x20time\x20without\x20wait\x20time.\n\n\r\n\x05\x04\x04\x02\x01\
    \x04\x12\x04>\x04=\x16\n\x0c\n\x05\x04\x04\x02\x01\x05\x12\x03>\x04\t\n\
    \x0c\n\x05\x04\x04\x02\x01\x01\x12\x03>\n\x14\n\x0c\n\x05\x04\x04\x02\
    \x01\x03\x12\x03>\x17\x18\n\n\n\x02\x04\x05\x12\x04A\0D\x01\n\n\n\x03\
    \x04\x05\x01\x12\x03A\x08\x10\n\x1a\n\x04\x04\x05\x02\0\x12\x03B\x04\x14\
    \"\r\x20total\x20count\n\n\r\n\x05\x04\x05\x02\0\x04\x12\x04B\x04A\x12\n\
    \x0c\n\x05\x04\x05\x02\0\x05\x12\x03B\x04\t\n\x0c\n\x05\x04\x05\x02\0\
    \x01\x12\x03B\n\x0f\n\x0c\n\x05\x04\x05\x02\0\x03\x12\x03B\x12\x13\n\x1e\
    \n\x04\x04\x05\x02\x01\x12\x03C\x04\x18\"\x11\x20processed\x20count\n\n\
    \r\n\x05\x04\x05\x02\x01\x04\x12\x04C\x04B\x14\n\x0c\n\x05\x04\x05\x02\
    \x01\x05\x12\x03C\x04\t\n\x0c\n\x05\x04\x05\x02\x01\x01\x12\x03C\n\x13\n\
    \x0c\n\x05\x04\x05\x02\x01\x03\x12\x03C\x16\x17\n\n\n\x02\x04\x06\x12\
    \x04F\0J\x01\n\n\n\x03\x04\x06\x01\x12\x03F\x08\x12\n\x0b\n\x04\x04\x06\
    \x02\0\x12\x03G\x04\x17\n\r\n\x05\x04\x06\x02\0\x04\x12\x04G\x04F\x14\n\
    \x0c\n\x05\x04\x06\x02\0\x06\x12\x03G\x04\x0c\n\x0c\n\x05\x04\x06\x02\0\
    \x01\x12\x03G\r\x12\n\x0c\n\x05\x04\x06\x02\0\x03\x12\x03G\x15\x16\n\x0b\
    \n\x04\x04\x06\x02\x01\x12\x03H\x04\x16\n\r\n\x05\x04\x06\x02\x01\x04\
    \x12\x04H\x04G\x17\n\x0c\n\x05\x04\x06\x02\x01\x06\x12\x03H\x04\x0c\n\
    \x0c\n\x05\x04\x06\x02\x01\x01\x12\x03H\r\x11\n\x0c\n\x05\x04\x06\x02\
    \x01\x03\x12\x03H\x14\x15\n\x0b\n\x04\x04\x06\x02\x02\x12\x03I\x04\x16\n\
    \r\n\x05\x04\x06\x02\x02\x04\x12\x04I\x04H\x16\n\x0c\n\x05\x04\x06\x02\
    \x02\x06\x12\x03I\x04\x0c\n\x0c\n\x05\x04\x06\x02\x02\x01\x12\x03I\r\x11\
    \n\x0c\n\x05\x04\x06\x02\x02\x03\x12\x03I\x14\x15\n\n\n\x02\x04\x07\x12\
    \x04L\0O\x01\n\n\n\x03\x04\x07\x01\x12\x03L\x08\x13\nA\n\x04\x04\x07\x02\
    \0\x12\x03M\x04\x1f\"4\x20set\x20when\x20ctx.handle_time\x20=\x20true\
    \x20or\x20meet\x20slow\x20query\n\n\r\n\x05\x04\x07\x02\0\x04\x12\x04M\
    \x04L\x15\n\x0c\n\x05\x04\x07\x02\0\x06\x12\x03M\x04\x0e\n\x0c\n\x05\x04\
    \x07\x02\0\x01\x12\x03M\x0f\x1a\n\x0c\n\x05\x04\x07\x02\0\x03\x12\x03M\
    \x1d\x1e\nA\n\x04\x04\x07\x02\x01\x12\x03N\x04\x1f\"4\x20set\x20when\x20\
    ctx.scan_detail\x20=\x20true\x20or\x20meet\x20slow\x20query\n\n\r\n\x05\
    \x04\x07\x02\x01\x04\x12\x04N\x04M\x1f\n\x0c\n\x05\x04\x07\x02\x01\x06\
    \x12\x03N\x04\x0e\n\x0c\n\x05\x04\x07\x02\x01\x01\x12\x03N\x0f\x1a\n\x0c\
    \n\x05\x04\x07\x02\x01\x03\x12\x03N\x1d\x1e\n\n\n\x02\x04\x08\x12\x04Q\0\
    U\x01\n\n\n\x03\x04\x08\x01\x12\x03Q\x08\x12\n\x0b\n\x04\x04\x08\x02\0\
    \x12\x03R\x04\x18\n\r\n\x05\x04\x08\x02\0\x04\x12\x04R\x04Q\x14\n\x0c\n\
    \x05\x04\x08\x02\0\x06\x12\x03R\x04\x0b\n\x0c\n\x05\x04\x08\x02\0\x01\
    \x12\x03R\x0c\x13\n\x0c\n\x05\x04\x08\x02\0\x03\x12\x03R\x16\x17\n\x0b\n\
    \x04\x04\x08\x02\x01\x12\x03S\x04\x12\n\r\n\x05\x04\x08\x02\x01\x04\x12\
    \x04S\x04R\x18\n\x0c\n\x05\x04\x08\x02\x01\x05\x12\x03S\x04\t\n\x0c\n\
    \x05\x04\x08\x02\x01\x01\x12\x03S\n\r\n\x0c\n\x05\x04\x08\x02\x01\x03\
    \x12\x03S\x10\x11\n\x0b\n\x04\x04\x08\x02\x02\x12\x03T\x04\x17\n\r\n\x05\
    \x04\x08\x02\x02\x04\x12\x04T\x04S\x12\n\x0c\n\x05\x04\x08\x02\x02\x05\
    \x12\x03T\x04\n\n\x0c\n\x05\x04\x08\x02\x02\x01\x12\x03T\x0b\x12\n\x0c\n\
    \x05\x04\x08\x02\x02\x03\x12\x03T\x15\x16\n\n\n\x02\x04\t\x12\x04W\0[\
    \x01\n\n\n\x03\x04\t\x01\x12\x03W\x08\x13\n\x0b\n\x04\x04\t\x02\0\x12\
    \x03X\x04#\n\r\n\x05\x04\t\x02\0\x04\x12\x04X\x04W\x15\n\x0c\n\x05\x04\t\
    \x02\0\x06\x12\x03X\x04\x11\n\x0c\n\x05\x04\t\x02\0\x01\x12\x03X\x12\x1e\
    \n\x0c\n\x05\x04\t\x02\0\x03\x12\x03X!\"\n\x0b\n\x04\x04\t\x02\x01\x12\
    \x03Y\x04\x17\n\r\n\x05\x04\t\x02\x01\x04\x12\x04Y\x04X#\n\x0c\n\x05\x04\
    \t\x02\x01\x06\x12\x03Y\x04\x0c\n\x0c\n\x05\x04\t\x02\x01\x01\x12\x03Y\r\
    \x12\n\x0c\n\x05\x04\t\x02\x01\x03\x12\x03Y\x15\x16\n\x0b\n\x04\x04\t\
    \x02\x02\x12\x03Z\x04\x14\n\r\n\x05\x04\t\x02\x02\x04\x12\x04Z\x04Y\x17\
    \n\x0c\n\x05\x04\t\x02\x02\x05\x12\x03Z\x04\t\n\x0c\n\x05\x04\t\x02\x02\
    \x01\x12\x03Z\n\x0f\n\x0c\n\x05\x04\t\x02\x02\x03\x12\x03Z\x12\x13\n\n\n\
    \x02\x04\n\x12\x04]\0d\x01\n\n\n\x03\x04\n\x01\x12\x03]\x08\x13\n\x0b\n\
    \x04\x04\n\x02\0\x12\x03^\x04\x18\n\r\n\x05\x04\n\x02\0\x04\x12\x04^\x04\
    ]\x15\n\x0c\n\x05\x04\n\x02\0\x06\x12\x03^\x04\x0b\n\x0c\n\x05\x04\n\x02\
    \0\x01\x12\x03^\x0c\x13\n\x0c\n\x05\x04\n\x02\0\x03\x12\x03^\x16\x17\n\
    \x0b\n\x04\x04\n\x02\x01\x12\x03_\x04\x18\n\r\n\x05\x04\n\x02\x01\x04\
    \x12\x04_\x04^\x18\n\x0c\n\x05\x04\n\x02\x01\x05\x12\x03_\x04\t\n\x0c\n\
    \x05\x04\n\x02\x01\x01\x12\x03_\n\x13\n\x0c\n\x05\x04\n\x02\x01\x03\x12\
    \x03_\x16\x17\n\x0b\n\x04\x04\n\x02\x02\x12\x03`\x04\x15\n\r\n\x05\x04\n\
    \x02\x02\x04\x12\x04`\x04_\x18\n\x0c\n\x05\x04\n\x02\x02\x05\x12\x03`\
    \x04\n\n\x0c\n\x05\x04\n\x02\x02\x01\x12\x03`\x0b\x10\n\x0c\n\x05\x04\n\
    \x02\x02\x03\x12\x03`\x13\x14\n\x0b\n\x04\x04\n\x02\x03\x12\x03a\x04\x17\
    \n\r\n\x05\x04\n\x02\x03\x04\x12\x04a\x04`\x15\n\x0c\n\x05\x04\n\x02\x03\
    \x05\x12\x03a\x04\n\n\x0c\n\x05\x04\n\x02\x03\x01\x12\x03a\x0b\x12\n\x0c\
    \n\x05\x04\n\x02\x03\x03\x12\x03a\x15\x16\n\x0b\n\x04\x04\n\x02\x04\x12\
    \x03b\x04\x16\n\r\n\x05\x04\n\x02\x04\x04\x12\x04b\x04a\x17\n\x0c\n\x05\
    \x04\n\x02\x04\x05\x12\x03b\x04\x08\n\x0c\n\x05\x04\n\x02\x04\x01\x12\
    \x03b\t\x11\n\x0c\n\x05\x04\n\x02\x04\x03\x12\x03b\x14\x15\n\x0b\n\x04\
    \x04\n\x02\x05\x12\x03c\x04\x15\n\r\n\x05\x04\n\x02\x05\x04\x12\x04c\x04\
    b\x16\n\x0c\n\x05\x04\n\x02\x05\x05\x12\x03c\x04\x08\n\x0c\n\x05\x04\n\
    \x02\x05\x01\x12\x03c\t\x10\n\x0c\n\x05\x04\n\x02\x05\x03\x12\x03c\x13\
    \x14\n\n\n\x02\x04\x0b\x12\x04f\0j\x01\n\n\n\x03\x04\x0b\x01\x12\x03f\
    \x08\x0e\n\x0b\n\x04\x04\x0b\x02\0\x12\x03g\x04\x17\n\r\n\x05\x04\x0b\
    \x02\0\x04\x12\x04g\x04f\x10\n\x0c\n\x05\x04\x0b\x02\0\x06\x12\x03g\x04\
    \x0c\n\x0c\n\x05\x04\x0b\x02\0\x01\x12\x03g\r\x12\n\x0c\n\x05\x04\x0b\
    \x02\0\x03\x12\x03g\x15\x16\n\x0b\n\x04\x04\x0b\x02\x01\x12\x03h\x04\x12\
    \n\r\n\x05\x04\x0b\x02\x01\x04\x12\x04h\x04g\x17\n\x0c\n\x05\x04\x0b\x02\
    \x01\x05\x12\x03h\x04\t\n\x0c\n\x05\x04\x0b\x02\x01\x01\x12\x03h\n\r\n\
    \x0c\n\x05\x04\x0b\x02\x01\x03\x12\x03h\x10\x11\n\x0b\n\x04\x04\x0b\x02\
    \x02\x12\x03i\x04\x14\n\r\n\x05\x04\x0b\x02\x02\x04\x12\x04i\x04h\x12\n\
    \x0c\n\x05\x04\x0b\x02\x02\x05\x12\x03i\x04\t\n\x0c\n\x05\x04\x0b\x02\
    \x02\x01\x12\x03i\n\x0f\n\x0c\n\x05\x04\x0b\x02\x02\x03\x12\x03i\x12\x13\
    \n\n\n\x02\x04\x0c\x12\x04l\0o\x01\n\n\n\x03\x04\x0c\x01\x12\x03l\x08\
    \x14\n\x0b\n\x04\x04\x0c\x02\0\x12\x03m\x04#\n\r\n\x05\x04\x0c\x02\0\x04\
    \x12\x04m\x04l\x16\n\x0c\n\x05\x04\x0c\x02\0\x06\x12\x03m\x04\x11\n\x0c\
    \n\x05\x04\x0c\x02\0\x01\x12\x03m\x12\x1e\n\x0c\n\x05\x04\x0c\x02\0\x03\
    \x12\x03m!\"\n\x0b\n\x04\x04\x0c\x02\x01\x12\x03n\x04\x1e\n\x0c\n\x05\
    \x04\x0c\x02\x01\x04\x12\x03n\x04\x0c\n\x0c\n\x05\x04\x0c\x02\x01\x06\
    \x12\x03n\r\x13\n\x0c\n\x05\x04\x0c\x02\x01\x01\x12\x03n\x14\x19\n\x0c\n\
    \x05\x04\x0c\x02\x01\x03\x12\x03n\x1c\x1d\n\n\n\x02\x05\x02\x12\x04q\0v\
    \x01\n\n\n\x03\x05\x02\x01\x12\x03q\x05\x07\n\x0b\n\x04\x05\x02\x02\0\
    \x12\x03r\x04\x0c\n\x0c\n\x05\x05\x02\x02\0\x01\x12\x03r\x04\x07\n\x0c\n\
    \x05\x05\x02\x02\0\x02\x12\x03r\n\x0b\n\x0b\n\x04\x05\x02\x02\x01\x12\
    \x03s\x04\x0c\n\x0c\n\x05\x05\x02\x02\x01\x01\x12\x03s\x04\x07\n\x0c\n\
    \x05\x05\x02\x02\x01\x02\x12\x03s\n\x0b\n\x0b\n\x04\x05\x02\x02\x02\x12\
    \x03t\x04\r\n\x0c\n\x05\x05\x02\x02\x02\x01\x12\x03t\x04\x08\n\x0c\n\x05\
    \x05\x02\x02\x02\x02\x12\x03t\x0b\x0c\n\x0b\n\x04\x05\x02\x02\x03\x12\
    \x03u\x04\x11\n\x0c\n\x05\x05\x02\x02\x03\x01\x12\x03u\x04\x0c\n\x0c\n\
    \x05\x05\x02\x02\x03\x02\x12\x03u\x0f\x10\n\n\n\x02\x04\r\x12\x04x\0|\
    \x01\n\n\n\x03\x04\r\x01\x12\x03x\x08\x10\n\x0b\n\x04\x04\r\x02\0\x12\
    \x03y\x04\x0e\n\r\n\x05\x04\r\x02\0\x04\x12\x04y\x04x\x12\n\x0c\n\x05\
    \x04\r\x02\0\x06\x12\x03y\x04\x06\n\x0c\n\x05\x04\r\x02\0\x01\x12\x03y\
    \x07\t\n\x0c\n\x05\x04\r\x02\0\x03\x12\x03y\x0c\r\n\x0b\n\x04\x04\r\x02\
    \x01\x12\x03z\x04\x12\n\r\n\x05\x04\r\x02\x01\x04\x12\x04z\x04y\x0e\n\
    \x0c\n\x05\x04\r\x02\x01\x05\x12\x03z\x04\t\n\x0c\n\x05\x04\r\x02\x01\
    \x01\x12\x03z\n\r\n\x0c\n\x05\x04\r\x02\x01\x03\x12\x03z\x10\x11\n\x0b\n\
    \x04\x04\r\x02\x02\x12\x03{\x04\x14\n\r\n\x05\x04\r\x02\x02\x04\x12\x04{\
    \x04z\x12\n\x0c\n\x05\x04\r\x02\x02\x05\x12\x03{\x04\t\n\x0c\n\x05\x04\r\
    \x02\x02\x01\x12\x03{\n\x0f\n\x0c\n\x05\x04\r\x02\x02\x03\x12\x03{\x12\
    \x13\n\x0b\n\x02\x04\x0e\x12\x05~\0\x86\x01\x01\n\n\n\x03\x04\x0e\x01\
    \x12\x03~\x08\x17\n\x0b\n\x04\x04\x0e\x02\0\x12\x03\x7f\x04\x18\n\r\n\
    \x05\x04\x0e\x02\0\x04\x12\x04\x7f\x04~\x19\n\x0c\n\x05\x04\x0e\x02\0\
    \x06\x12\x03\x7f\x04\x0b\n\x0c\n\x05\x04\x0e\x02\0\x01\x12\x03\x7f\x0c\
    \x13\n\x0c\n\x05\x04\x0e\x02\0\x03\x12\x03\x7f\x16\x17\n\x0c\n\x04\x04\
    \x0e\x02\x01\x12\x04\x80\x01\x04$\n\r\n\x05\x04\x0e\x02\x01\x04\x12\x04\
    \x80\x01\x04\x0c\n\r\n\x05\x04\x0e\x02\x01\x06\x12\x04\x80\x01\r\x15\n\r\
    \n\x05\x04\x0e\x02\x01\x01\x12\x04\x80\x01\x16\x1f\n\r\n\x05\x04\x0e\x02\
    \x01\x03\x12\x04\x80\x01\"#\n\x20\n\x04\x04\x0e\x02\x02\x12\x04\x82\x01\
    \x04\x1b\x1a\x12\x20primary_lock_key\n\n\x0f\n\x05\x04\x0e\x02\x02\x04\
    \x12\x06\x82\x01\x04\x80\x01$\n\r\n\x05\x04\x0e\x02\x02\x05\x12\x04\x82\
    \x01\x04\t\n\r\n\x05\x04\x0e\x02\x02\x01\x12\x04\x82\x01\n\x16\n\r\n\x05\
    \x04\x0e\x02\x02\x03\x12\x04\x82\x01\x19\x1a\n\x0c\n\x04\x04\x0e\x02\x03\
    \x12\x04\x83\x01\x04\x1d\n\x0f\n\x05\x04\x0e\x02\x03\x04\x12\x06\x83\x01\
    \x04\x82\x01\x1b\n\r\n\x05\x04\x0e\x02\x03\x05\x12\x04\x83\x01\x04\n\n\r\
    \n\x05\x04\x0e\x02\x03\x01\x12\x04\x83\x01\x0b\x18\n\r\n\x05\x04\x0e\x02\
    \x03\x03\x12\x04\x83\x01\x1b\x1c\n\x0c\n\x04\x04\x0e\x02\x04\x12\x04\x84\
    \x01\x04\x18\n\x0f\n\x05\x04\x0e\x02\x04\x04\x12\x06\x84\x01\x04\x83\x01\
    \x1d\n\r\n\x05\x04\x0e\x02\x04\x05\x12\x04\x84\x01\x04\n\n\r\n\x05\x04\
    \x0e\x02\x04\x01\x12\x04\x84\x01\x0b\x13\n\r\n\x05\x04\x0e\x02\x04\x03\
    \x12\x04\x84\x01\x16\x17\n\x0c\n\x04\x04\x0e\x02\x05\x12\x04\x85\x01\x04\
    #\n\x0f\n\x05\x04\x0e\x02\x05\x04\x12\x06\x85\x01\x04\x84\x01\x18\n\r\n\
    \x05\x04\x0e\x02\x05\x05\x12\x04\x85\x01\x04\x08\n\r\n\x05\x04\x0e\x02\
    \x05\x01\x12\x04\x85\x01\t\x1e\n\r\n\x05\x04\x0e\x02\x05\x03\x12\x04\x85\
    \x01!\"\n\x0c\n\x02\x04\x0f\x12\x06\x88\x01\0\x8b\x01\x01\n\x0b\n\x03\
    \x04\x0f\x01\x12\x04\x88\x01\x08\x18\n\x0c\n\x04\x04\x0f\x02\0\x12\x04\
    \x89\x01\x04#\n\x0f\n\x05\x04\x0f\x02\0\x04\x12\x06\x89\x01\x04\x88\x01\
    \x1a\n\r\n\x05\x04\x0f\x02\0\x06\x12\x04\x89\x01\x04\x11\n\r\n\x05\x04\
    \x0f\x02\0\x01\x12\x04\x89\x01\x12\x1e\n\r\n\x05\x04\x0f\x02\0\x03\x12\
    \x04\x89\x01!\"\n\x0c\n\x04\x04\x0f\x02\x01\x12\x04\x8a\x01\x04!\n\r\n\
    \x05\x04\x0f\x02\x01\x04\x12\x04\x8a\x01\x04\x0c\n\r\n\x05\x04\x0f\x02\
    \x01\x06\x12\x04\x8a\x01\r\x15\n\r\n\x05\x04\x0f\x02\x01\x01\x12\x04\x8a\
    \x01\x16\x1c\n\r\n\x05\x04\x0f\x02\x01\x03\x12\x04\x8a\x01\x1f\x20\n\x0c\
    \n\x02\x04\x10\x12\x06\x8d\x01\0\x94\x01\x01\n\x0b\n\x03\x04\x10\x01\x12\
    \x04\x8d\x01\x08\x15\n\x0b\n\x03\x04\x10\t\x12\x04\x8e\x01\r\x0f\n\x0c\n\
    \x04\x04\x10\t\0\x12\x04\x8e\x01\r\x0e\n\r\n\x05\x04\x10\t\0\x01\x12\x04\
    \x8e\x01\r\x0e\n\r\n\x05\x04\x10\t\0\x02\x12\x04\x8e\x01\r\x0e\n\x0b\n\
    \x03\x04\x10\n\x12\x04\x8f\x01\r\x16\n\x0c\n\x04\x04\x10\n\0\x12\x04\x8f\
    \x01\r\x15\n\x0c\n\x04\x04\x10\x02\0\x12\x04\x90\x01\x04\x18\n\x0f\n\x05\
    \x04\x10\x02\0\x04\x12\x06\x90\x01\x04\x8f\x01\x16\n\r\n\x05\x04\x10\x02\
    \0\x06\x12\x04\x90\x01\x04\x0b\n\r\n\x05\x04\x10\x02\0\x01\x12\x04\x90\
    \x01\x0c\x13\n\r\n\x05\x04\x10\x02\0\x03\x12\x04\x90\x01\x16\x17\n\x0c\n\
    \x04\x04\x10\x02\x01\x12\x04\x91\x01\x04\x1d\n\x0f\n\x05\x04\x10\x02\x01\
    \x04\x12\x06\x91\x01\x04\x90\x01\x18\n\r\n\x05\x04\x10\x02\x01\x05\x12\
    \x04\x91\x01\x04\n\n\r\n\x05\x04\x10\x02\x01\x01\x12\x04\x91\x01\x0b\x18\
    \n\r\n\x05\x04\x10\x02\x01\x03\x12\x04\x91\x01\x1b\x1c\n\x0c\n\x04\x04\
    \x10\x02\x02\x12\x04\x92\x01\x04\x1c\n\r\n\x05\x04\x10\x02\x02\x04\x12\
    \x04\x92\x01\x04\x0c\n\r\n\x05\x04\x10\x02\x02\x05\x12\x04\x92\x01\r\x12\
    \n\r\n\x05\x04\x10\x02\x02\x01\x12\x04\x92\x01\x13\x17\n\r\n\x05\x04\x10\
    \x02\x02\x03\x12\x04\x92\x01\x1a\x1b\n\x0c\n\x04\x04\x10\x02\x03\x12\x04\
    \x93\x01\x04\x1e\n\x0f\n\x05\x04\x10\x02\x03\x04\x12\x06\x93\x01\x04\x92\
    \x01\x1c\n\r\n\x05\x04\x10\x02\x03\x05\x12\x04\x93\x01\x04\n\n\r\n\x05\
    \x04\x10\x02\x03\x01\x12\x04\x93\x01\x0b\x19\n\r\n\x05\x04\x10\x02\x03\
    \x03\x12\x04\x93\x01\x1c\x1d\n\x0c\n\x02\x04\x11\x12\x06\x96\x01\0\x99\
    \x01\x01\n\x0b\n\x03\x04\x11\x01\x12\x04\x96\x01\x08\x16\n\x0c\n\x04\x04\
    \x11\x02\0\x12\x04\x97\x01\x04#\n\x0f\n\x05\x04\x11\x02\0\x04\x12\x06\
    \x97\x01\x04\x96\x01\x18\n\r\n\x05\x04\x11\x02\0\x06\x12\x04\x97\x01\x04\
    \x11\n\r\n\x05\x04\x11\x02\0\x01\x12\x04\x97\x01\x12\x1e\n\r\n\x05\x04\
    \x11\x02\0\x03\x12\x04\x97\x01!\"\n\x0c\n\x04\x04\x11\x02\x01\x12\x04\
    \x98\x01\x04\x17\n\x0f\n\x05\x04\x11\x02\x01\x04\x12\x06\x98\x01\x04\x97\
    \x01#\n\r\n\x05\x04\x11\x02\x01\x06\x12\x04\x98\x01\x04\x0c\n\r\n\x05\
    \x04\x11\x02\x01\x01\x12\x04\x98\x01\r\x12\n\r\n\x05\x04\x11\x02\x01\x03\
    \x12\x04\x98\x01\x15\x16\n\x0c\n\x02\x04\x12\x12\x06\x9b\x01\0\x9e\x01\
    \x01\n\x0b\n\x03\x04\x12\x01\x12\x04\x9b\x01\x08\x15\n\x0c\n\x04\x04\x12\
    \x02\0\x12\x04\x9c\x01\x04$\n\r\n\x05\x04\x12\x02\0\x04\x12\x04\x9c\x01\
    \x04\x0c\n\r\n\x05\x04\x12\x02\0\x06\x12\x04\x9c\x01\r\x15\n\r\n\x05\x04\
    \x12\x02\0\x01\x12\x04\x9c\x01\x16\x1f\n\r\n\x05\x04\x12\x02\0\x03\x12\
    \x04\x9c\x01\"#\n\x0c\n\x04\x04\x12\x02\x01\x12\x04\x9d\x01\x04\x1e\n\
    \x0f\n\x05\x04\x12\x02\x01\x04\x12\x06\x9d\x01\x04\x9c\x01$\n\r\n\x05\
    \x04\x12\x02\x01\x05\x12\x04\x9d\x01\x04\n\n\r\n\x05\x04\x12\x02\x01\x01\
    \x12\x04\x9d\x01\x0b\x19\n\r\n\x05\x04\x12\x02\x01\x03\x12\x04\x9d\x01\
    \x1c\x1d\n\x0c\n\x02\x04\x13\x12\x06\xa0\x01\0\xa3\x01\x01\n\x0b\n\x03\
    \x04\x13\x01\x12\x04\xa0\x01\x08\x16\n\x0c\n\x04\x04\x13\x02\0\x12\x04\
    \xa1\x01\x04#\n\x0f\n\x05\x04\x13\x02\0\x04\x12\x06\xa1\x01\x04\xa0\x01\
    \x18\n\r\n\x05\x04\x13\x02\0\x06\x12\x04\xa1\x01\x04\x11\n\r\n\x05\x04\
    \x13\x02\0\x01\x12\x04\xa1\x01\x12\x1e\n\r\n\x05\x04\x13\x02\0\x03\x12\
    \x04\xa1\x01!\"\n\x0c\n\x04\x04\x13\x02\x01\x12\x04\xa2\x01\x04\x15\n\
    \x0f\n\x05\x04\x13\x02\x01\x04\x12\x06\xa2\x01\x04\xa1\x01#\n\r\n\x05\
    \x04\x13\x02\x01\x05\x12\x04\xa2\x01\x04\n\n\r\n\x05\x04\x13\x02\x01\x01\
    \x12\x04\xa2\x01\x0b\x10\n\r\n\x05\x04\x13\x02\x01\x03\x12\x04\xa2\x01\
    \x13\x14\n\x0c\n\x02\x04\x14\x12\x06\xa5\x01\0\xa9\x01\x01\n\x0b\n\x03\
    \x04\x14\x01\x12\x04\xa5\x01\x08\x1c\n\x0c\n\x04\x04\x14\x02\0\x12\x04\
    \xa6\x01\x04\x18\n\x0f\n\x05\x04\x14\x02\0\x04\x12\x06\xa6\x01\x04\xa5\
    \x01\x1e\n\r\n\x05\x04\x14\x02\0\x06\x12\x04\xa6\x01\x04\x0b\n\r\n\x05\
    \x04\x14\x02\0\x01\x12\x04\xa6\x01\x0c\x13\n\r\n\x05\x04\x14\x02\0\x03\
    \x12\x04\xa6\x01\x16\x17\n\x0c\n\x04\x04\x14\x02\x01\x12\x04\xa7\x01\x04\
    \x1d\n\x0f\n\x05\x04\x14\x02\x01\x04\x12\x06\xa7\x01\x04\xa6\x01\x18\n\r\
    \n\x05\x04\x14\x02\x01\x05\x12\x04\xa7\x01\x04\n\n\r\n\x05\x04\x14\x02\
    \x01\x01\x12\x04\xa7\x01\x0b\x18\n\r\n\x05\x04\x14\x02\x01\x03\x12\x04\
    \xa7\x01\x1b\x1c\n\x0c\n\x04\x04\x14\x02\x02\x12\x04\xa8\x01\x04\x1c\n\r\
    \n\x05\x04\x14\x02\x02\x04\x12\x04\xa8\x01\x04\x0c\n\r\n\x05\x04\x14\x02\
    \x02\x05\x12\x04\xa8\x01\r\x12\n\r\n\x05\x04\x14\x02\x02\x01\x12\x04\xa8\
    \x01\x13\x17\n\r\n\x05\x04\x14\x02\x02\x03\x12\x04\xa8\x01\x1a\x1b\n\x0c\
    \n\x02\x04\x15\x12\x06\xab\x01\0\xae\x01\x01\n\x0b\n\x03\x04\x15\x01\x12\
    \x04\xab\x01\x08\x1d\n\x0c\n\x04\x04\x15\x02\0\x12\x04\xac\x01\x04#\n\
    \x0f\n\x05\x04\x15\x02\0\x04\x12\x06\xac\x01\x04\xab\x01\x1f\n\r\n\x05\
    \x04\x15\x02\0\x06\x12\x04\xac\x01\x04\x11\n\r\n\x05\x04\x15\x02\0\x01\
    \x12\x04\xac\x01\x12\x1e\n\r\n\x05\x04\x15\x02\0\x03\x12\x04\xac\x01!\"\
    \n\x0c\n\x04\x04\x15\x02\x01\x12\x04\xad\x01\x04\x17\n\x0f\n\x05\x04\x15\
    \x02\x01\x04\x12\x06\xad\x01\x04\xac\x01#\n\r\n\x05\x04\x15\x02\x01\x06\
    \x12\x04\xad\x01\x04\x0c\n\r\n\x05\x04\x15\x02\x01\x01\x12\x04\xad\x01\r\
    \x12\n\r\n\x05\x04\x15\x02\x01\x03\x12\x04\xad\x01\x15\x16\n\x0c\n\x02\
    \x04\x16\x12\x06\xb0\x01\0\xb4\x01\x01\n\x0b\n\x03\x04\x16\x01\x12\x04\
    \xb0\x01\x08\x16\n\x0c\n\x04\x04\x16\x02\0\x12\x04\xb1\x01\x04\x18\n\x0f\
    \n\x05\x04\x16\x02\0\x04\x12\x06\xb1\x01\x04\xb0\x01\x18\n\r\n\x05\x04\
    \x16\x02\0\x06\x12\x04\xb1\x01\x04\x0b\n\r\n\x05\x04\x16\x02\0\x01\x12\
    \x04\xb1\x01\x0c\x13\n\r\n\x05\x04\x16\x02\0\x03\x12\x04\xb1\x01\x16\x17\
    \n\x0c\n\x04\x04\x16\x02\x01\x12\x04\xb2\x01\x04\x12\n\x0f\n\x05\x04\x16\
    \x02\x01\x04\x12\x06\xb2\x01\x04\xb1\x01\x18\n\r\n\x05\x04\x16\x02\x01\
    \x05\x12\x04\xb2\x01\x04\t\n\r\n\x05\x04\x16\x02\x01\x01\x12\x04\xb2\x01\
    \n\r\n\r\n\x05\x04\x16\x02\x01\x03\x12\x04\xb2\x01\x10\x11\n\x0c\n\x04\
    \x04\x16\x02\x02\x12\x04\xb3\x01\x04\x1d\n\x0f\n\x05\x04\x16\x02\x02\x04\
    \x12\x06\xb3\x01\x04\xb2\x01\x12\n\r\n\x05\x04\x16\x02\x02\x05\x12\x04\
    \xb3\x01\x04\n\n\r\n\x05\x04\x16\x02\x02\x01\x12\x04\xb3\x01\x0b\x18\n\r\
    \n\x05\x04\x16\x02\x02\x03\x12\x04\xb3\x01\x1b\x1c\n\x0c\n\x02\x04\x17\
    \x12\x06\xb6\x01\0\xba\x01\x01\n\x0b\n\x03\x04\x17\x01\x12\x04\xb6\x01\
    \x08\x17\n\x0c\n\x04\x04\x17\x02\0\x12\x04\xb7\x01\x04#\n\x0f\n\x05\x04\
    \x17\x02\0\x04\x12\x06\xb7\x01\x04\xb6\x01\x19\n\r\n\x05\x04\x17\x02\0\
    \x06\x12\x04\xb7\x01\x04\x11\n\r\n\x05\x04\x17\x02\0\x01\x12\x04\xb7\x01\
    \x12\x1e\n\r\n\x05\x04\x17\x02\0\x03\x12\x04\xb7\x01!\"\n\x0c\n\x04\x04\
    \x17\x02\x01\x12\x04\xb8\x01\x04\x17\n\x0f\n\x05\x04\x17\x02\x01\x04\x12\
    \x06\xb8\x01\x04\xb7\x01#\n\r\n\x05\x04\x17\x02\x01\x06\x12\x04\xb8\x01\
    \x04\x0c\n\r\n\x05\x04\x17\x02\x01\x01\x12\x04\xb8\x01\r\x12\n\r\n\x05\
    \x04\x17\x02\x01\x03\x12\x04\xb8\x01\x15\x16\n8\n\x04\x04\x17\x02\x02\
    \x12\x04\xb9\x01\x04\x1e\"*\x20set\x20this\x20if\x20the\x20key\x20is\x20\
    already\x20committed\n\n\x0f\n\x05\x04\x17\x02\x02\x04\x12\x06\xb9\x01\
    \x04\xb8\x01\x17\n\r\n\x05\x04\x17\x02\x02\x05\x12\x04\xb9\x01\x04\n\n\r\
    \n\x05\x04\x17\x02\x02\x01\x12\x04\xb9\x01\x0b\x19\n\r\n\x05\x04\x17\x02\
    \x02\x03\x12\x04\xb9\x01\x1c\x1d\n\x0c\n\x02\x04\x18\x12\x06\xbc\x01\0\
    \xc0\x01\x01\n\x0b\n\x03\x04\x18\x01\x12\x04\xbc\x01\x08\x17\n\x0c\n\x04\
    \x04\x18\x02\0\x12\x04\xbd\x01\x04\x18\n\x0f\n\x05\x04\x18\x02\0\x04\x12\
    \x06\xbd\x01\x04\xbc\x01\x19\n\r\n\x05\x04\x18\x02\0\x06\x12\x04\xbd\x01\
    \x04\x0b\n\r\n\x05\x04\x18\x02\0\x01\x12\x04\xbd\x01\x0c\x13\n\r\n\x05\
    \x04\x18\x02\0\x03\x12\x04\xbd\x01\x16\x17\n\x0c\n\x04\x04\x18\x02\x01\
    \x12\x04\xbe\x01\x04\x1c\n\r\n\x05\x04\x18\x02\x01\x04\x12\x04\xbe\x01\
    \x04\x0c\n\r\n\x05\x04\x18\x02\x01\x05\x12\x04\xbe\x01\r\x12\n\r\n\x05\
    \x04\x18\x02\x01\x01\x12\x04\xbe\x01\x13\x17\n\r\n\x05\x04\x18\x02\x01\
    \x03\x12\x04\xbe\x01\x1a\x1b\n\x0c\n\x04\x04\x18\x02\x02\x12\x04\xbf\x01\
    \x04\x17\n\x0f\n\x05\x04\x18\x02\x02\x04\x12\x06\xbf\x01\x04\xbe\x01\x1c\
    \n\r\n\x05\x04\x18\x02\x02\x05\x12\x04\xbf\x01\x04\n\n\r\n\x05\x04\x18\
    \x02\x02\x01\x12\x04\xbf\x01\x0b\x12\n\r\n\x05\x04\x18\x02\x02\x03\x12\
    \x04\xbf\x01\x15\x16\n\x0c\n\x02\x04\x19\x12\x06\xc2\x01\0\xc5\x01\x01\n\
    \x0b\n\x03\x04\x19\x01\x12\x04\xc2\x01\x08\x18\n\x0c\n\x04\x04\x19\x02\0\
    \x12\x04\xc3\x01\x04#\n\x0f\n\x05\x04\x19\x02\0\x04\x12\x06\xc3\x01\x04\
    \xc2\x01\x1a\n\r\n\x05\x04\x19\x02\0\x06\x12\x04\xc3\x01\x04\x11\n\r\n\
    \x05\x04\x19\x02\0\x01\x12\x04\xc3\x01\x12\x1e\n\r\n\x05\x04\x19\x02\0\
    \x03\x12\x04\xc3\x01!\"\n\x0c\n\x04\x04\x19\x02\x01\x12\x04\xc4\x01\x04\
    \x1e\n\r\n\x05\x04\x19\x02\x01\x04\x12\x04\xc4\x01\x04\x0c\n\r\n\x05\x04\
    \x19\x02\x01\x06\x12\x04\xc4\x01\r\x13\n\r\n\x05\x04\x19\x02\x01\x01\x12\
    \x04\xc4\x01\x14\x19\n\r\n\x05\x04\x19\x02\x01\x03\x12\x04\xc4\x01\x1c\
    \x1d\n\x0c\n\x02\x04\x1a\x12\x06\xc7\x01\0\xcc\x01\x01\n\x0b\n\x03\x04\
    \x1a\x01\x12\x04\xc7\x01\x08\x17\n\x0c\n\x04\x04\x1a\x02\0\x12\x04\xc8\
    \x01\x04\x18\n\x0f\n\x05\x04\x1a\x02\0\x04\x12\x06\xc8\x01\x04\xc7\x01\
    \x19\n\r\n\x05\x04\x1a\x02\0\x06\x12\x04\xc8\x01\x04\x0b\n\r\n\x05\x04\
    \x1a\x02\0\x01\x12\x04\xc8\x01\x0c\x13\n\r\n\x05\x04\x1a\x02\0\x03\x12\
    \x04\xc8\x01\x16\x17\n\x0c\n\x04\x04\x1a\x02\x01\x12\x04\xc9\x01\x04\x1b\
    \n\x0f\n\x05\x04\x1a\x02\x01\x04\x12\x06\xc9\x01\x04\xc8\x01\x18\n\r\n\
    \x05\x04\x1a\x02\x01\x05\x12\x04\xc9\x01\x04\n\n\r\n\x05\x04\x1a\x02\x01\
    \x01\x12\x04\xc9\x01\x0b\x16\n\r\n\x05\x04\x1a\x02\x01\x03\x12\x04\xc9\
    \x01\x19\x1a\n\x0c\n\x04\x04\x1a\x02\x02\x12\x04\xca\x01\x04\x18\n\x0f\n\
    \x05\x04\x1a\x02\x02\x04\x12\x06\xca\x01\x04\xc9\x01\x1b\n\r\n\x05\x04\
    \x1a\x02\x02\x05\x12\x04\xca\x01\x04\t\n\r\n\x05\x04\x1a\x02\x02\x01\x12\
    \x04\xca\x01\n\x13\n\r\n\x05\x04\x1a\x02\x02\x03\x12\x04\xca\x01\x16\x17\
    \n\x0c\n\x04\x04\x1a\x02\x03\x12\x04\xcb\x01\x04\x15\n\x0f\n\x05\x04\x1a\
    \x02\x03\x04\x12\x06\xcb\x01\x04\xca\x01\x18\n\r\n\x05\x04\x1a\x02\x03\
    \x05\x12\x04\xcb\x01\x04\n\n\r\n\x05\x04\x1a\x02\x03\x01\x12\x04\xcb\x01\
    \x0b\x10\n\r\n\x05\x04\x1a\x02\x03\x03\x12\x04\xcb\x01\x13\x14\n\x0c\n\
    \x02\x04\x1b\x12\x06\xce\x01\0\xd2\x01\x01\n\x0b\n\x03\x04\x1b\x01\x12\
    \x04\xce\x01\x08\x18\n\x0c\n\x04\x04\x1b\x02\0\x12\x04\xcf\x01\x04#\n\
    \x0f\n\x05\x04\x1b\x02\0\x04\x12\x06\xcf\x01\x04\xce\x01\x1a\n\r\n\x05\
    \x04\x1b\x02\0\x06\x12\x04\xcf\x01\x04\x11\n\r\n\x05\x04\x1b\x02\0\x01\
    \x12\x04\xcf\x01\x12\x1e\n\r\n\x05\x04\x1b\x02\0\x03\x12\x04\xcf\x01!\"\
    \n\x0c\n\x04\x04\x1b\x02\x01\x12\x04\xd0\x01\x04\x17\n\x0f\n\x05\x04\x1b\
    \x02\x01\x04\x12\x06\xd0\x01\x04\xcf\x01#\n\r\n\x05\x04\x1b\x02\x01\x06\
    \x12\x04\xd0\x01\x04\x0c\n\r\n\x05\x04\x1b\x02\x01\x01\x12\x04\xd0\x01\r\
    \x12\n\r\n\x05\x04\x1b\x02\x01\x03\x12\x04\xd0\x01\x15\x16\n\x0c\n\x04\
    \x04\x1b\x02\x02\x12\x04\xd1\x01\x04\x20\n\r\n\x05\x04\x1b\x02\x02\x04\
    \x12\x04\xd1\x01\x04\x0c\n\r\n\x05\x04\x1b\x02\x02\x06\x12\x04\xd1\x01\r\
    \x15\n\r\n\x05\x04\x1b\x02\x02\x01\x12\x04\xd1\x01\x16\x1b\n\r\n\x05\x04\
    \x1b\x02\x02\x03\x12\x04\xd1\x01\x1e\x1f\n\x0c\n\x02\x04\x1c\x12\x06\xd4\
    \x01\0\xd7\x01\x01\n\x0b\n\x03\x04\x1c\x01\x12\x04\xd4\x01\x08\x0f\n\x0c\
    \n\x04\x04\x1c\x02\0\x12\x04\xd5\x01\x04\x13\n\x0f\n\x05\x04\x1c\x02\0\
    \x04\x12\x06\xd5\x01\x04\xd4\x01\x11\n\r\n\x05\x04\x1c\x02\0\x05\x12\x04\
    \xd5\x01\x04\n\n\r\n\x05\x04\x1c\x02\0\x01\x12\x04\xd5\x01\x0b\x0e\n\r\n\
    \x05\x04\x1c\x02\0\x03\x12\x04\xd5\x01\x11\x12\n\x0c\n\x04\x04\x1c\x02\
    \x01\x12\x04\xd6\x01\x04\x16\n\x0f\n\x05\x04\x1c\x02\x01\x04\x12\x06\xd6\
    \x01\x04\xd5\x01\x13\n\r\n\x05\x04\x1c\x02\x01\x05\x12\x04\xd6\x01\x04\n\
    \n\r\n\x05\x04\x1c\x02\x01\x01\x12\x04\xd6\x01\x0b\x11\n\r\n\x05\x04\x1c\
    \x02\x01\x03\x12\x04\xd6\x01\x14\x15\n\x0c\n\x02\x04\x1d\x12\x06\xd9\x01\
    \0\xdf\x01\x01\n\x0b\n\x03\x04\x1d\x01\x12\x04\xd9\x01\x08\x1a\n\x0c\n\
    \x04\x04\x1d\x02\0\x12\x04\xda\x01\x04\x18\n\x0f\n\x05\x04\x1d\x02\0\x04\
    \x12\x06\xda\x01\x04\xd9\x01\x1c\n\r\n\x05\x04\x1d\x02\0\x06\x12\x04\xda\
    \x01\x04\x0b\n\r\n\x05\x04\x1d\x02\0\x01\x12\x04\xda\x01\x0c\x13\n\r\n\
    \x05\x04\x1d\x02\0\x03\x12\x04\xda\x01\x16\x17\n\x0c\n\x04\x04\x1d\x02\
    \x01\x12\x04\xdb\x01\x04\x1e\n\x0f\n\x05\x04\x1d\x02\x01\x04\x12\x06\xdb\
    \x01\x04\xda\x01\x18\n\r\n\x05\x04\x1d\x02\x01\x05\x12\x04\xdb\x01\x04\n\
    \n\r\n\x05\x04\x1d\x02\x01\x01\x12\x04\xdb\x01\x0b\x18\n\r\n\x05\x04\x1d\
    \x02\x01\x03\x12\x04\xdb\x01\x1c\x1d\n9\n\x04\x04\x1d\x02\x02\x12\x04\
    \xdd\x01\x04\x1e\x1a+\x20If\x20the\x20txn\x20is\x20rolled\x20back,\x20do\
    \x20not\x20set\x20it.\n\n\x0f\n\x05\x04\x1d\x02\x02\x04\x12\x06\xdd\x01\
    \x04\xdb\x01\x1e\n\r\n\x05\x04\x1d\x02\x02\x05\x12\x04\xdd\x01\x04\n\n\r\
    \n\x05\x04\x1d\x02\x02\x01\x12\x04\xdd\x01\x0b\x19\n\r\n\x05\x04\x1d\x02\
    \x02\x03\x12\x04\xdd\x01\x1c\x1d\n\x0c\n\x04\x04\x1d\x02\x03\x12\x04\xde\
    \x01\x04#\n\r\n\x05\x04\x1d\x02\x03\x04\x12\x04\xde\x01\x04\x0c\n\r\n\
    \x05\x04\x1d\x02\x03\x06\x12\x04\xde\x01\r\x14\n\r\n\x05\x04\x1d\x02\x03\
    \x01\x12\x04\xde\x01\x15\x1e\n\r\n\x05\x04\x1d\x02\x03\x03\x12\x04\xde\
    \x01!\"\n\x0c\n\x02\x04\x1e\x12\x06\xe1\x01\0\xe4\x01\x01\n\x0b\n\x03\
    \x04\x1e\x01\x12\x04\xe1\x01\x08\x1b\n\x0c\n\x04\x04\x1e\x02\0\x12\x04\
    \xe2\x01\x04#\n\x0f\n\x05\x04\x1e\x02\0\x04\x12\x06\xe2\x01\x04\xe1\x01\
    \x1d\n\r\n\x05\x04\x1e\x02\0\x06\x12\x04\xe2\x01\x04\x11\n\r\n\x05\x04\
    \x1e\x02\0\x01\x12\x04\xe2\x01\x12\x1e\n\r\n\x05\x04\x1e\x02\0\x03\x12\
    \x04\xe2\x01!\"\n\x0c\n\x04\x04\x1e\x02\x01\x12\x04\xe3\x01\x04\x17\n\
    \x0f\n\x05\x04\x1e\x02\x01\x04\x12\x06\xe3\x01\x04\xe2\x01#\n\r\n\x05\
    \x04\x1e\x02\x01\x06\x12\x04\xe3\x01\x04\x0c\n\r\n\x05\x04\x1e\x02\x01\
    \x01\x12\x04\xe3\x01\r\x12\n\r\n\x05\x04\x1e\x02\x01\x03\x12\x04\xe3\x01\
    \x15\x16\n\x0c\n\x02\x04\x1f\x12\x06\xe6\x01\0\xe9\x01\x01\n\x0b\n\x03\
    \x04\x1f\x01\x12\x04\xe6\x01\x08\x11\n\x0c\n\x04\x04\x1f\x02\0\x12\x04\
    \xe7\x01\x04\x18\n\x0f\n\x05\x04\x1f\x02\0\x04\x12\x06\xe7\x01\x04\xe6\
    \x01\x13\n\r\n\x05\x04\x1f\x02\0\x06\x12\x04\xe7\x01\x04\x0b\n\r\n\x05\
    \x04\x1f\x02\0\x01\x12\x04\xe7\x01\x0c\x13\n\r\n\x05\x04\x1f\x02\0\x03\
    \x12\x04\xe7\x01\x16\x17\n\x0c\n\x04\x04\x1f\x02\x01\x12\x04\xe8\x01\x04\
    \x1a\n\x0f\n\x05\x04\x1f\x02\x01\x04\x12\x06\xe8\x01\x04\xe7\x01\x18\n\r\
    \n\x05\x04\x1f\x02\x01\x05\x12\x04\xe8\x01\x04\n\n\r\n\x05\x04\x1f\x02\
    \x01\x01\x12\x04\xe8\x01\x0b\x15\n\r\n\x05\x04\x1f\x02\x01\x03\x12\x04\
    \xe8\x01\x18\x19\n\x0c\n\x02\x04\x20\x12\x06\xeb\x01\0\xee\x01\x01\n\x0b\
    \n\x03\x04\x20\x01\x12\x04\xeb\x01\x08\x12\n\x0c\n\x04\x04\x20\x02\0\x12\
    \x04\xec\x01\x04#\n\x0f\n\x05\x04\x20\x02\0\x04\x12\x06\xec\x01\x04\xeb\
    \x01\x14\n\r\n\x05\x04\x20\x02\0\x06\x12\x04\xec\x01\x04\x11\n\r\n\x05\
    \x04\x20\x02\0\x01\x12\x04\xec\x01\x12\x1e\n\r\n\x05\x04\x20\x02\0\x03\
    \x12\x04\xec\x01!\"\n\x0c\n\x04\x04\x20\x02\x01\x12\x04\xed\x01\x04\x17\
    \n\x0f\n\x05\x04\x20\x02\x01\x04\x12\x06\xed\x01\x04\xec\x01#\n\r\n\x05\
    \x04\x20\x02\x01\x06\x12\x04\xed\x01\x04\x0c\n\r\n\x05\x04\x20\x02\x01\
    \x01\x12\x04\xed\x01\r\x12\n\r\n\x05\x04\x20\x02\x01\x03\x12\x04\xed\x01\
    \x15\x16\n\x0c\n\x02\x04!\x12\x06\xf0\x01\0\xf4\x01\x01\n\x0b\n\x03\x04!\
    \x01\x12\x04\xf0\x01\x08\x15\n\x0c\n\x04\x04!\x02\0\x12\x04\xf1\x01\x04\
    \x18\n\x0f\n\x05\x04!\x02\0\x04\x12\x06\xf1\x01\x04\xf0\x01\x17\n\r\n\
    \x05\x04!\x02\0\x06\x12\x04\xf1\x01\x04\x0b\n\r\n\x05\x04!\x02\0\x01\x12\
    \x04\xf1\x01\x0c\x13\n\r\n\x05\x04!\x02\0\x03\x12\x04\xf1\x01\x16\x17\n\
    \x0c\n\x04\x04!\x02\x01\x12\x04\xf2\x01\x04\x12\n\x0f\n\x05\x04!\x02\x01\
    \x04\x12\x06\xf2\x01\x04\xf1\x01\x18\n\r\n\x05\x04!\x02\x01\x05\x12\x04\
    \xf2\x01\x04\t\n\r\n\x05\x04!\x02\x01\x01\x12\x04\xf2\x01\n\r\n\r\n\x05\
    \x04!\x02\x01\x03\x12\x04\xf2\x01\x10\x11\n\x0c\n\x04\x04!\x02\x02\x12\
    \x04\xf3\x01\x04\x12\n\x0f\n\x05\x04!\x02\x02\x04\x12\x06\xf3\x01\x04\
    \xf2\x01\x12\n\r\n\x05\x04!\x02\x02\x05\x12\x04\xf3\x01\x04\n\n\r\n\x05\
    \x04!\x02\x02\x01\x12\x04\xf3\x01\x0b\r\n\r\n\x05\x04!\x02\x02\x03\x12\
    \x04\xf3\x01\x10\x11\n\x0c\n\x02\x04\"\x12\x06\xf6\x01\0\xfa\x01\x01\n\
    \x0b\n\x03\x04\"\x01\x12\x04\xf6\x01\x08\x16\n\x0c\n\x04\x04\"\x02\0\x12\
    \x04\xf7\x01\x04#\n\x0f\n\x05\x04\"\x02\0\x04\x12\x06\xf7\x01\x04\xf6\
    \x01\x18\n\r\n\x05\x04\"\x02\0\x06\x12\x04\xf7\x01\x04\x11\n\r\n\x05\x04\
    \"\x02\0\x01\x12\x04\xf7\x01\x12\x1e\n\r\n\x05\x04\"\x02\0\x03\x12\x04\
    \xf7\x01!\"\n\x0c\n\x04\x04\"\x02\x01\x12\x04\xf8\x01\x04\x15\n\x0f\n\
    \x05\x04\"\x02\x01\x04\x12\x06\xf8\x01\x04\xf7\x01#\n\r\n\x05\x04\"\x02\
    \x01\x05\x12\x04\xf8\x01\x04\n\n\r\n\x05\x04\"\x02\x01\x01\x12\x04\xf8\
    \x01\x0b\x10\n\r\n\x05\x04\"\x02\x01\x03\x12\x04\xf8\x01\x13\x14\n\x0c\n\
    \x04\x04\"\x02\x02\x12\x04\xf9\x01\x04\x14\n\x0f\n\x05\x04\"\x02\x02\x04\
    \x12\x06\xf9\x01\x04\xf8\x01\x15\n\r\n\x05\x04\"\x02\x02\x05\x12\x04\xf9\
    \x01\x04\t\n\r\n\x05\x04\"\x02\x02\x01\x12\x04\xf9\x01\n\x0f\n\r\n\x05\
    \x04\"\x02\x02\x03\x12\x04\xf9\x01\x12\x13\n\x0c\n\x02\x04#\x12\x06\xfc\
    \x01\0\x81\x02\x01\n\x0b\n\x03\x04#\x01\x12\x04\xfc\x01\x08\x15\n\x0c\n\
    \x04\x04#\x02\0\x12\x04\xfd\x01\x04\x18\n\x0f\n\x05\x04#\x02\0\x04\x12\
    \x06\xfd\x01\x04\xfc\x01\x17\n\r\n\x05\x04#\x02\0\x06\x12\x04\xfd\x01\
    \x04\x0b\n\r\n\x05\x04#\x02\0\x01\x12\x04\xfd\x01\x0c\x13\n\r\n\x05\x04#\
    \x02\0\x03\x12\x04\xfd\x01\x16\x17\n\x0c\n\x04\x04#\x02\x01\x12\x04\xfe\
    \x01\x04\x12\n\x0f\n\x05\x04#\x02\x01\x04\x12\x06\xfe\x01\x04\xfd\x01\
    \x18\n\r\n\x05\x04#\x02\x01\x05\x12\x04\xfe\x01\x04\t\n\r\n\x05\x04#\x02\
    \x01\x01\x12\x04\xfe\x01\n\r\n\r\n\x05\x04#\x02\x01\x03\x12\x04\xfe\x01\
    \x10\x11\n\x0c\n\x04\x04#\x02\x02\x12\x04\xff\x01\x04\x14\n\x0f\n\x05\
    \x04#\x02\x02\x04\x12\x06\xff\x01\x04\xfe\x01\x12\n\r\n\x05\x04#\x02\x02\
    \x05\x12\x04\xff\x01\x04\t\n\r\n\x05\x04#\x02\x02\x01\x12\x04\xff\x01\n\
    \x0f\n\r\n\x05\x04#\x02\x02\x03\x12\x04\xff\x01\x12\x13\n\x0c\n\x04\x04#\
    \x02\x03\x12\x04\x80\x02\x04\x12\n\x0f\n\x05\x04#\x02\x03\x04\x12\x06\
    \x80\x02\x04\xff\x01\x14\n\r\n\x05\x04#\x02\x03\x05\x12\x04\x80\x02\x04\
    \n\n\r\n\x05\x04#\x02\x03\x01\x12\x04\x80\x02\x0b\r\n\r\n\x05\x04#\x02\
    \x03\x03\x12\x04\x80\x02\x10\x11\n\x0c\n\x02\x04$\x12\x06\x83\x02\0\x86\
    \x02\x01\n\x0b\n\x03\x04$\x01\x12\x04\x83\x02\x08\x16\n\x0c\n\x04\x04$\
    \x02\0\x12\x04\x84\x02\x04#\n\x0f\n\x05\x04$\x02\0\x04\x12\x06\x84\x02\
    \x04\x83\x02\x18\n\r\n\x05\x04$\x02\0\x06\x12\x04\x84\x02\x04\x11\n\r\n\
    \x05\x04$\x02\0\x01\x12\x04\x84\x02\x12\x1e\n\r\n\x05\x04$\x02\0\x03\x12\
    \x04\x84\x02!\"\n\x0c\n\x04\x04$\x02\x01\x12\x04\x85\x02\x04\x15\n\x0f\n\
    \x05\x04$\x02\x01\x04\x12\x06\x85\x02\x04\x84\x02#\n\r\n\x05\x04$\x02\
    \x01\x05\x12\x04\x85\x02\x04\n\n\r\n\x05\x04$\x02\x01\x01\x12\x04\x85\
    \x02\x0b\x10\n\r\n\x05\x04$\x02\x01\x03\x12\x04\x85\x02\x13\x14\n\x0c\n\
    \x02\x04%\x12\x06\x88\x02\0\x8c\x02\x01\n\x0b\n\x03\x04%\x01\x12\x04\x88\
    \x02\x08\x1a\n\x0c\n\x04\x04%\x02\0\x12\x04\x89\x02\x04\x18\n\x0f\n\x05\
    \x04%\x02\0\x04\x12\x06\x89\x02\x04\x88\x02\x1c\n\r\n\x05\x04%\x02\0\x06\
    \x12\x04\x89\x02\x04\x0b\n\r\n\x05\x04%\x02\0\x01\x12\x04\x89\x02\x0c\
    \x13\n\r\n\x05\x04%\x02\0\x03\x12\x04\x89\x02\x16\x17\n\x0c\n\x04\x04%\
    \x02\x01\x12\x04\x8a\x02\x04\x1e\n\r\n\x05\x04%\x02\x01\x04\x12\x04\x8a\
    \x02\x04\x0c\n\r\n\x05\x04%\x02\x01\x06\x12\x04\x8a\x02\r\x13\n\r\n\x05\
    \x04%\x02\x01\x01\x12\x04\x8a\x02\x14\x19\n\r\n\x05\x04%\x02\x01\x03\x12\
    \x04\x8a\x02\x1c\x1d\n\x0c\n\x04\x04%\x02\x02\x12\x04\x8b\x02\x04\x12\n\
    \x0f\n\x05\x04%\x02\x02\x04\x12\x06\x8b\x02\x04\x8a\x02\x1e\n\r\n\x05\
    \x04%\x02\x02\x05\x12\x04\x8b\x02\x04\n\n\r\n\x05\x04%\x02\x02\x01\x12\
    \x04\x8b\x02\x0b\r\n\r\n\x05\x04%\x02\x02\x03\x12\x04\x8b\x02\x10\x11\n\
    \x0c\n\x02\x04&\x12\x06\x8e\x02\0\x91\x02\x01\n\x0b\n\x03\x04&\x01\x12\
    \x04\x8e\x02\x08\x1b\n\x0c\n\x04\x04&\x02\0\x12\x04\x8f\x02\x04#\n\x0f\n\
    \x05\x04&\x02\0\x04\x12\x06\x8f\x02\x04\x8e\x02\x1d\n\r\n\x05\x04&\x02\0\
    \x06\x12\x04\x8f\x02\x04\x11\n\r\n\x05\x04&\x02\0\x01\x12\x04\x8f\x02\
    \x12\x1e\n\r\n\x05\x04&\x02\0\x03\x12\x04\x8f\x02!\"\n\x0c\n\x04\x04&\
    \x02\x01\x12\x04\x90\x02\x04\x15\n\x0f\n\x05\x04&\x02\x01\x04\x12\x06\
    \x90\x02\x04\x8f\x02#\n\r\n\x05\x04&\x02\x01\x05\x12\x04\x90\x02\x04\n\n\
    \r\n\x05\x04&\x02\x01\x01\x12\x04\x90\x02\x0b\x10\n\r\n\x05\x04&\x02\x01\
    \x03\x12\x04\x90\x02\x13\x14\n\x0c\n\x02\x04'\x12\x06\x93\x02\0\x97\x02\
    \x01\n\x0b\n\x03\x04'\x01\x12\x04\x93\x02\x08\x1a\n\x0c\n\x04\x04'\x02\0\
    \x12\x04\x94\x02\x04\x18\n\x0f\n\x05\x04'\x02\0\x04\x12\x06\x94\x02\x04\
    \x93\x02\x1c\n\r\n\x05\x04'\x02\0\x06\x12\x04\x94\x02\x04\x0b\n\r\n\x05\
    \x04'\x02\0\x01\x12\x04\x94\x02\x0c\x13\n\r\n\x05\x04'\x02\0\x03\x12\x04\
    \x94\x02\x16\x17\n\x0c\n\x04\x04'\x02\x01\x12\x04\x95\x02\x04\x1c\n\r\n\
    \x05\x04'\x02\x01\x04\x12\x04\x95\x02\x04\x0c\n\r\n\x05\x04'\x02\x01\x05\
    \x12\x04\x95\x02\r\x12\n\r\n\x05\x04'\x02\x01\x01\x12\x04\x95\x02\x13\
    \x17\n\r\n\x05\x04'\x02\x01\x03\x12\x04\x95\x02\x1a\x1b\n\x0c\n\x04\x04'\
    \x02\x02\x12\x04\x96\x02\x04\x12\n\x0f\n\x05\x04'\x02\x02\x04\x12\x06\
    \x96\x02\x04\x95\x02\x1c\n\r\n\x05\x04'\x02\x02\x05\x12\x04\x96\x02\x04\
    \n\n\r\n\x05\x04'\x02\x02\x01\x12\x04\x96\x02\x0b\r\n\r\n\x05\x04'\x02\
    \x02\x03\x12\x04\x96\x02\x10\x11\n\x0c\n\x02\x04(\x12\x06\x99\x02\0\x9c\
    \x02\x01\n\x0b\n\x03\x04(\x01\x12\x04\x99\x02\x08\x1b\n\x0c\n\x04\x04(\
    \x02\0\x12\x04\x9a\x02\x04#\n\x0f\n\x05\x04(\x02\0\x04\x12\x06\x9a\x02\
    \x04\x99\x02\x1d\n\r\n\x05\x04(\x02\0\x06\x12\x04\x9a\x02\x04\x11\n\r\n\
    \x05\x04(\x02\0\x01\x12\x04\x9a\x02\x12\x1e\n\r\n\x05\x04(\x02\0\x03\x12\
    \x04\x9a\x02!\"\n\x0c\n\x04\x04(\x02\x01\x12\x04\x9b\x02\x04\x1e\n\r\n\
    \x05\x04(\x02\x01\x04\x12\x04\x9b\x02\x04\x0c\n\r\n\x05\x04(\x02\x01\x06\
    \x12\x04\x9b\x02\r\x13\n\r\n\x05\x04(\x02\x01\x01\x12\x04\x9b\x02\x14\
    \x19\n\r\n\x05\x04(\x02\x01\x03\x12\x04\x9b\x02\x1c\x1d\n\x0c\n\x02\x04)\
    \x12\x06\x9e\x02\0\xa2\x02\x01\n\x0b\n\x03\x04)\x01\x12\x04\x9e\x02\x08\
    \x18\n\x0c\n\x04\x04)\x02\0\x12\x04\x9f\x02\x04\x18\n\x0f\n\x05\x04)\x02\
    \0\x04\x12\x06\x9f\x02\x04\x9e\x02\x1a\n\r\n\x05\x04)\x02\0\x06\x12\x04\
    \x9f\x02\x04\x0b\n\r\n\x05\x04)\x02\0\x01\x12\x04\x9f\x02\x0c\x13\n\r\n\
    \x05\x04)\x02\0\x03\x12\x04\x9f\x02\x16\x17\n\x0c\n\x04\x04)\x02\x01\x12\
    \x04\xa0\x02\x04\x12\n\x0f\n\x05\x04)\x02\x01\x04\x12\x06\xa0\x02\x04\
    \x9f\x02\x18\n\r\n\x05\x04)\x02\x01\x05\x12\x04\xa0\x02\x04\t\n\r\n\x05\
    \x04)\x02\x01\x01\x12\x04\xa0\x02\n\r\n\r\n\x05\x04)\x02\x01\x03\x12\x04\
    \xa0\x02\x10\x11\n\x0c\n\x04\x04)\x02\x02\x12\x04\xa1\x02\x04\x12\n\x0f\
    \n\x05\x04)\x02\x02\x04\x12\x06\xa1\x02\x04\xa0\x02\x12\n\r\n\x05\x04)\
    \x02\x02\x05\x12\x04\xa1\x02\x04\n\n\r\n\x05\x04)\x02\x02\x01\x12\x04\
    \xa1\x02\x0b\r\n\r\n\x05\x04)\x02\x02\x03\x12\x04\xa1\x02\x10\x11\n\x0c\
    \n\x02\x04*\x12\x06\xa4\x02\0\xa7\x02\x01\n\x0b\n\x03\x04*\x01\x12\x04\
    \xa4\x02\x08\x19\n\x0c\n\x04\x04*\x02\0\x12\x04\xa5\x02\x04#\n\x0f\n\x05\
    \x04*\x02\0\x04\x12\x06\xa5\x02\x04\xa4\x02\x1b\n\r\n\x05\x04*\x02\0\x06\
    \x12\x04\xa5\x02\x04\x11\n\r\n\x05\x04*\x02\0\x01\x12\x04\xa5\x02\x12\
    \x1e\n\r\n\x05\x04*\x02\0\x03\x12\x04\xa5\x02!\"\n\x0c\n\x04\x04*\x02\
    \x01\x12\x04\xa6\x02\x04\x15\n\x0f\n\x05\x04*\x02\x01\x04\x12\x06\xa6\
    \x02\x04\xa5\x02#\n\r\n\x05\x04*\x02\x01\x05\x12\x04\xa6\x02\x04\n\n\r\n\
    \x05\x04*\x02\x01\x01\x12\x04\xa6\x02\x0b\x10\n\r\n\x05\x04*\x02\x01\x03\
    \x12\x04\xa6\x02\x13\x14\n\x0c\n\x02\x04+\x12\x06\xa9\x02\0\xad\x02\x01\
    \n\x0b\n\x03\x04+\x01\x12\x04\xa9\x02\x08\x1d\n\x0c\n\x04\x04+\x02\0\x12\
    \x04\xaa\x02\x04\x18\n\x0f\n\x05\x04+\x02\0\x04\x12\x06\xaa\x02\x04\xa9\
    \x02\x1f\n\r\n\x05\x04+\x02\0\x06\x12\x04\xaa\x02\x04\x0b\n\r\n\x05\x04+\
    \x02\0\x01\x12\x04\xaa\x02\x0c\x13\n\r\n\x05\x04+\x02\0\x03\x12\x04\xaa\
    \x02\x16\x17\n\x0c\n\x04\x04+\x02\x01\x12\x04\xab\x02\x04\x1c\n\r\n\x05\
    \x04+\x02\x01\x04\x12\x04\xab\x02\x04\x0c\n\r\n\x05\x04+\x02\x01\x05\x12\
    \x04\xab\x02\r\x12\n\r\n\x05\x04+\x02\x01\x01\x12\x04\xab\x02\x13\x17\n\
    \r\n\x05\x04+\x02\x01\x03\x12\x04\xab\x02\x1a\x1b\n\x0c\n\x04\x04+\x02\
    \x02\x12\x04\xac\x02\x04\x12\n\x0f\n\x05\x04+\x02\x02\x04\x12\x06\xac\
    \x02\x04\xab\x02\x1c\n\r\n\x05\x04+\x02\x02\x05\x12\x04\xac\x02\x04\n\n\
    \r\n\x05\x04+\x02\x02\x01\x12\x04\xac\x02\x0b\r\n\r\n\x05\x04+\x02\x02\
    \x03\x12\x04\xac\x02\x10\x11\n\x0c\n\x02\x04,\x12\x06\xaf\x02\0\xb2\x02\
    \x01\n\x0b\n\x03\x04,\x01\x12\x04\xaf\x02\x08\x1e\n\x0c\n\x04\x04,\x02\0\
    \x12\x04\xb0\x02\x04#\n\x0f\n\x05\x04,\x02\0\x04\x12\x06\xb0\x02\x04\xaf\
    \x02\x20\n\r\n\x05\x04,\x02\0\x06\x12\x04\xb0\x02\x04\x11\n\r\n\x05\x04,\
    \x02\0\x01\x12\x04\xb0\x02\x12\x1e\n\r\n\x05\x04,\x02\0\x03\x12\x04\xb0\
    \x02!\"\n\x0c\n\x04\x04,\x02\x01\x12\x04\xb1\x02\x04\x15\n\x0f\n\x05\x04\
    ,\x02\x01\x04\x12\x06\xb1\x02\x04\xb0\x02#\n\r\n\x05\x04,\x02\x01\x05\
    \x12\x04\xb1\x02\x04\n\n\r\n\x05\x04,\x02\x01\x01\x12\x04\xb1\x02\x0b\
    \x10\n\r\n\x05\x04,\x02\x01\x03\x12\x04\xb1\x02\x13\x14\n\x0c\n\x02\x04-\
    \x12\x06\xb4\x02\0\xb8\x02\x01\n\x0b\n\x03\x04-\x01\x12\x04\xb4\x02\x08\
    \x1a\n\x0c\n\x04\x04-\x02\0\x12\x04\xb5\x02\x04\x18\n\x0f\n\x05\x04-\x02\
    \0\x04\x12\x06\xb5\x02\x04\xb4\x02\x1c\n\r\n\x05\x04-\x02\0\x06\x12\x04\
    \xb5\x02\x04\x0b\n\r\n\x05\x04-\x02\0\x01\x12\x04\xb5\x02\x0c\x13\n\r\n\
    \x05\x04-\x02\0\x03\x12\x04\xb5\x02\x16\x17\n\x0c\n\x04\x04-\x02\x01\x12\
    \x04\xb6\x02\x04\x18\n\x0f\n\x05\x04-\x02\x01\x04\x12\x06\xb6\x02\x04\
    \xb5\x02\x18\n\r\n\x05\x04-\x02\x01\x05\x12\x04\xb6\x02\x04\t\n\r\n\x05\
    \x04-\x02\x01\x01\x12\x04\xb6\x02\n\x13\n\r\n\x05\x04-\x02\x01\x03\x12\
    \x04\xb6\x02\x16\x17\n\x0c\n\x04\x04-\x02\x02\x12\x04\xb7\x02\x04\x16\n\
    \x0f\n\x05\x04-\x02\x02\x04\x12\x06\xb7\x02\x04\xb6\x02\x18\n\r\n\x05\
    \x04-\x02\x02\x05\x12\x04\xb7\x02\x04\t\n\r\n\x05\x04-\x02\x02\x01\x12\
    \x04\xb7\x02\n\x11\n\r\n\x05\x04-\x02\x02\x03\x12\x04\xb7\x02\x14\x15\n\
    \x0c\n\x02\x04.\x12\x06\xba\x02\0\xbd\x02\x01\n\x0b\n\x03\x04.\x01\x12\
    \x04\xba\x02\x08\x1b\n\x0c\n\x04\x04.\x02\0\x12\x04\xbb\x02\x04#\n\x0f\n\
    \x05\x04.\x02\0\x04\x12\x06\xbb\x02\x04\xba\x02\x1d\n\r\n\x05\x04.\x02\0\
    \x06\x12\x04\xbb\x02\x04\x11\n\r\n\x05\x04.\x02\0\x01\x12\x04\xbb\x02\
    \x12\x1e\n\r\n\x05\x04.\x02\0\x03\x12\x04\xbb\x02!\"\n\x0c\n\x04\x04.\
    \x02\x01\x12\x04\xbc\x02\x04\x15\n\x0f\n\x05\x04.\x02\x01\x04\x12\x06\
    \xbc\x02\x04\xbb\x02#\n\r\n\x05\x04.\x02\x01\x05\x12\x04\xbc\x02\x04\n\n\
    \r\n\x05\x04.\x02\x01\x01\x12\x04\xbc\x02\x0b\x10\n\r\n\x05\x04.\x02\x01\
    \x03\x12\x04\xbc\x02\x13\x14\n\x0c\n\x02\x04/\x12\x06\xbf\x02\0\xc4\x02\
    \x01\n\x0b\n\x03\x04/\x01\x12\x04\xbf\x02\x08\x1d\n\x0c\n\x04\x04/\x02\0\
    \x12\x04\xc0\x02\x04\x18\n\x0f\n\x05\x04/\x02\0\x04\x12\x06\xc0\x02\x04\
    \xbf\x02\x1f\n\r\n\x05\x04/\x02\0\x06\x12\x04\xc0\x02\x04\x0b\n\r\n\x05\
    \x04/\x02\0\x01\x12\x04\xc0\x02\x0c\x13\n\r\n\x05\x04/\x02\0\x03\x12\x04\
    \xc0\x02\x16\x17\n\x0c\n\x04\x04/\x02\x01\x12\x04\xc1\x02\x04\x18\n\x0f\
    \n\x05\x04/\x02\x01\x04\x12\x06\xc1\x02\x04\xc0\x02\x18\n\r\n\x05\x04/\
    \x02\x01\x05\x12\x04\xc1\x02\x04\t\n\r\n\x05\x04/\x02\x01\x01\x12\x04\
    \xc1\x02\n\x13\n\r\n\x05\x04/\x02\x01\x03\x12\x04\xc1\x02\x16\x17\n\x0c\
    \n\x04\x04/\x02\x02\x12\x04\xc2\x02\x04\x16\n\x0f\n\x05\x04/\x02\x02\x04\
    \x12\x06\xc2\x02\x04\xc1\x02\x18\n\r\n\x05\x04/\x02\x02\x05\x12\x04\xc2\
    \x02\x04\t\n\r\n\x05\x04/\x02\x02\x01\x12\x04\xc2\x02\n\x11\n\r\n\x05\
    \x04/\x02\x02\x03\x12\x04\xc2\x02\x14\x15\n\x0c\n\x04\x04/\x02\x03\x12\
    \x04\xc3\x02\x04\x12\n\x0f\n\x05\x04/\x02\x03\x04\x12\x06\xc3\x02\x04\
    \xc2\x02\x16\n\r\n\x05\x04/\x02\x03\x05\x12\x04\xc3\x02\x04\n\n\r\n\x05\
    \x04/\x02\x03\x01\x12\x04\xc3\x02\x0b\r\n\r\n\x05\x04/\x02\x03\x03\x12\
    \x04\xc3\x02\x10\x11\n\x0c\n\x02\x040\x12\x06\xc6\x02\0\xc9\x02\x01\n\
    \x0b\n\x03\x040\x01\x12\x04\xc6\x02\x08\x1e\n\x0c\n\x04\x040\x02\0\x12\
    \x04\xc7\x02\x04#\n\x0f\n\x05\x040\x02\0\x04\x12\x06\xc7\x02\x04\xc6\x02\
    \x20\n\r\n\x05\x040\x02\0\x06\x12\x04\xc7\x02\x04\x11\n\r\n\x05\x040\x02\
    \0\x01\x12\x04\xc7\x02\x12\x1e\n\r\n\x05\x040\x02\0\x03\x12\x04\xc7\x02!\
    \"\n\x0c\n\x04\x040\x02\x01\x12\x04\xc8\x02\x04\x15\n\x0f\n\x05\x040\x02\
    \x01\x04\x12\x06\xc8\x02\x04\xc7\x02#\n\r\n\x05\x040\x02\x01\x05\x12\x04\
    \xc8\x02\x04\n\n\r\n\x05\x040\x02\x01\x01\x12\x04\xc8\x02\x0b\x10\n\r\n\
    \x05\x040\x02\x01\x03\x12\x04\xc8\x02\x13\x14\n\x0c\n\x02\x041\x12\x06\
    \xcb\x02\0\xd1\x02\x01\n\x0b\n\x03\x041\x01\x12\x04\xcb\x02\x08\x16\n\
    \x0c\n\x04\x041\x02\0\x12\x04\xcc\x02\x04\x18\n\x0f\n\x05\x041\x02\0\x04\
    \x12\x06\xcc\x02\x04\xcb\x02\x18\n\r\n\x05\x041\x02\0\x06\x12\x04\xcc\
    \x02\x04\x0b\n\r\n\x05\x041\x02\0\x01\x12\x04\xcc\x02\x0c\x13\n\r\n\x05\
    \x041\x02\0\x03\x12\x04\xcc\x02\x16\x17\n\x0c\n\x04\x041\x02\x01\x12\x04\
    \xcd\x02\x04\x18\n\x0f\n\x05\x041\x02\x01\x04\x12\x06\xcd\x02\x04\xcc\
    \x02\x18\n\r\n\x05\x041\x02\x01\x05\x12\x04\xcd\x02\x04\t\n\r\n\x05\x041\
    \x02\x01\x01\x12\x04\xcd\x02\n\x13\n\r\n\x05\x041\x02\x01\x03\x12\x04\
    \xcd\x02\x16\x17\n\x0c\n\x04\x041\x02\x02\x12\x04\xce\x02\x04\x15\n\x0f\
    \n\x05\x041\x02\x02\x04\x12\x06\xce\x02\x04\xcd\x02\x18\n\r\n\x05\x041\
    \x02\x02\x05\x12\x04\xce\x02\x04\n\n\r\n\x05\x041\x02\x02\x01\x12\x04\
    \xce\x02\x0b\x10\n\r\n\x05\x041\x02\x02\x03\x12\x04\xce\x02\x13\x14\n\
    \x0c\n\x04\x041\x02\x03\x12\x04\xcf\x02\x04\x16\n\x0f\n\x05\x041\x02\x03\
    \x04\x12\x06\xcf\x02\x04\xce\x02\x15\n\r\n\x05\x041\x02\x03\x05\x12\x04\
    \xcf\x02\x04\x08\n\r\n\x05\x041\x02\x03\x01\x12\x04\xcf\x02\t\x11\n\r\n\
    \x05\x041\x02\x03\x03\x12\x04\xcf\x02\x14\x15\n\x0c\n\x04\x041\x02\x04\
    \x12\x04\xd0\x02\x04\x12\n\x0f\n\x05\x041\x02\x04\x04\x12\x06\xd0\x02\
    \x04\xcf\x02\x16\n\r\n\x05\x041\x02\x04\x05\x12\x04\xd0\x02\x04\n\n\r\n\
    \x05\x041\x02\x04\x01\x12\x04\xd0\x02\x0b\r\n\r\n\x05\x041\x02\x04\x03\
    \x12\x04\xd0\x02\x10\x11\n\x0c\n\x02\x042\x12\x06\xd3\x02\0\xd6\x02\x01\
    \n\x0b\n\x03\x042\x01\x12\x04\xd3\x02\x08\x17\n\x0c\n\x04\x042\x02\0\x12\
    \x04\xd4\x02\x04#\n\x0f\n\x05\x042\x02\0\x04\x12\x06\xd4\x02\x04\xd3\x02\
    \x19\n\r\n\x05\x042\x02\0\x06\x12\x04\xd4\x02\x04\x11\n\r\n\x05\x042\x02\
    \0\x01\x12\x04\xd4\x02\x12\x1e\n\r\n\x05\x042\x02\0\x03\x12\x04\xd4\x02!\
    \"\n\x0c\n\x04\x042\x02\x01\x12\x04\xd5\x02\x04\x1c\n\r\n\x05\x042\x02\
    \x01\x04\x12\x04\xd5\x02\x04\x0c\n\r\n\x05\x042\x02\x01\x06\x12\x04\xd5\
    \x02\r\x13\n\r\n\x05\x042\x02\x01\x01\x12\x04\xd5\x02\x14\x17\n\r\n\x05\
    \x042\x02\x01\x03\x12\x04\xd5\x02\x1a\x1b\n\x0c\n\x02\x043\x12\x06\xd8\
    \x02\0\xdb\x02\x01\n\x0b\n\x03\x043\x01\x12\x04\xd8\x02\x08\x10\n\x0c\n\
    \x04\x043\x02\0\x12\x04\xd9\x02\x02\x16\n\x0f\n\x05\x043\x02\0\x04\x12\
    \x06\xd9\x02\x02\xd8\x02\x12\n\r\n\x05\x043\x02\0\x05\x12\x04\xd9\x02\
    \x02\x07\n\r\n\x05\x043\x02\0\x01\x12\x04\xd9\x02\x08\x11\n\r\n\x05\x043\
    \x02\0\x03\x12\x04\xd9\x02\x14\x15\n\x0c\n\x04\x043\x02\x01\x12\x04\xda\
    \x02\x02\x14\n\x0f\n\x05\x043\x02\x01\x04\x12\x06\xda\x02\x02\xd9\x02\
    \x16\n\r\n\x05\x043\x02\x01\x05\x12\x04\xda\x02\x02\x07\n\r\n\x05\x043\
    \x02\x01\x01\x12\x04\xda\x02\x08\x0f\n\r\n\x05\x043\x02\x01\x03\x12\x04\
    \xda\x02\x12\x13\n\x0c\n\x02\x044\x12\x06\xdd\x02\0\xe3\x02\x01\n\x0b\n\
    \x03\x044\x01\x12\x04\xdd\x02\x08\x1b\n\x0c\n\x04\x044\x02\0\x12\x04\xde\
    \x02\x04\x18\n\x0f\n\x05\x044\x02\0\x04\x12\x06\xde\x02\x04\xdd\x02\x1d\
    \n\r\n\x05\x044\x02\0\x06\x12\x04\xde\x02\x04\x0b\n\r\n\x05\x044\x02\0\
    \x01\x12\x04\xde\x02\x0c\x13\n\r\n\x05\x044\x02\0\x03\x12\x04\xde\x02\
    \x16\x17\n\x1e\n\x04\x044\x02\x01\x12\x04\xdf\x02\x04!\"\x10\x20scanning\
    \x20range\n\n\r\n\x05\x044\x02\x01\x04\x12\x04\xdf\x02\x04\x0c\n\r\n\x05\
    \x044\x02\x01\x06\x12\x04\xdf\x02\r\x15\n\r\n\x05\x044\x02\x01\x01\x12\
    \x04\xdf\x02\x16\x1c\n\r\n\x05\x044\x02\x01\x03\x12\x04\xdf\x02\x1f\x20\
    \nH\n\x04\x044\x02\x02\x12\x04\xe0\x02\x04\x1a\":\x20max\x20number\x20of\
    \x20returning\x20kv\x20pairs\x20for\x20each\x20scanning\x20range\n\n\x0f\
    \n\x05\x044\x02\x02\x04\x12\x06\xe0\x02\x04\xdf\x02!\n\r\n\x05\x044\x02\
    \x02\x05\x12\x04\xe0\x02\x04\n\n\r\n\x05\x044\x02\x02\x01\x12\x04\xe0\
    \x02\x0b\x15\n\r\n\x05\x044\x02\x02\x03\x12\x04\xe0\x02\x18\x19\n\x0c\n\
    \x04\x044\x02\x03\x12\x04\xe1\x02\x04\x16\n\x0f\n\x05\x044\x02\x03\x04\
    \x12\x06\xe1\x02\x04\xe0\x02\x1a\n\r\n\x05\x044\x02\x03\x05\x12\x04\xe1\
    \x02\x04\x08\n\r\n\x05\x044\x02\x03\x01\x12\x04\xe1\x02\t\x11\n\r\n\x05\
    \x044\x02\x03\x03\x12\x04\xe1\x02\x14\x15\n\x0c\n\x04\x044\x02\x04\x12\
    \x04\xe2\x02\x04\x12\n\x0f\n\x05\x044\x02\x04\x04\x12\x06\xe2\x02\x04\
    \xe1\x02\x16\n\r\n\x05\x044\x02\x04\x05\x12\x04\xe2\x02\x04\n\n\r\n\x05\
    \x044\x02\x04\x01\x12\x04\xe2\x02\x0b\r\n\r\n\x05\x044\x02\x04\x03\x12\
    \x04\xe2\x02\x10\x11\n\x0c\n\x02\x045\x12\x06\xe5\x02\0\xe8\x02\x01\n\
    \x0b\n\x03\x045\x01\x12\x04\xe5\x02\x08\x1c\n\x0c\n\x04\x045\x02\0\x12\
    \x04\xe6\x02\x04#\n\x0f\n\x05\x045\x02\0\x04\x12\x06\xe6\x02\x04\xe5\x02\
    \x1e\n\r\n\x05\x045\x02\0\x06\x12\x04\xe6\x02\x04\x11\n\r\n\x05\x045\x02\
    \0\x01\x12\x04\xe6\x02\x12\x1e\n\r\n\x05\x045\x02\0\x03\x12\x04\xe6\x02!\
    \"\n\x0c\n\x04\x045\x02\x01\x12\x04\xe7\x02\x04\x1c\n\r\n\x05\x045\x02\
    \x01\x04\x12\x04\xe7\x02\x04\x0c\n\r\n\x05\x045\x02\x01\x06\x12\x04\xe7\
    \x02\r\x13\n\r\n\x05\x045\x02\x01\x01\x12\x04\xe7\x02\x14\x17\n\r\n\x05\
    \x045\x02\x01\x03\x12\x04\xe7\x02\x1a\x1b\n\x0c\n\x02\x046\x12\x06\xea\
    \x02\0\xef\x02\x01\n\x0b\n\x03\x046\x01\x12\x04\xea\x02\x08\x11\n\x0c\n\
    \x04\x046\x02\0\x12\x04\xeb\x02\x04\x10\n\x0f\n\x05\x046\x02\0\x04\x12\
    \x06\xeb\x02\x04\xea\x02\x13\n\r\n\x05\x046\x02\0\x06\x12\x04\xeb\x02\
    \x04\x06\n\r\n\x05\x046\x02\0\x01\x12\x04\xeb\x02\x07\x0b\n\r\n\x05\x046\
    \x02\0\x03\x12\x04\xeb\x02\x0e\x0f\n\x0c\n\x04\x046\x02\x01\x12\x04\xec\
    \x02\x04\x18\n\x0f\n\x05\x046\x02\x01\x04\x12\x06\xec\x02\x04\xeb\x02\
    \x10\n\r\n\x05\x046\x02\x01\x05\x12\x04\xec\x02\x04\n\n\r\n\x05\x046\x02\
    \x01\x01\x12\x04\xec\x02\x0b\x13\n\r\n\x05\x046\x02\x01\x03\x12\x04\xec\
    \x02\x16\x17\n\x0c\n\x04\x046\x02\x02\x12\x04\xed\x02\x04\x19\n\x0f\n\
    \x05\x046\x02\x02\x04\x12\x06\xed\x02\x04\xec\x02\x18\n\r\n\x05\x046\x02\
    \x02\x05\x12\x04\xed\x02\x04\n\n\r\n\x05\x046\x02\x02\x01\x12\x04\xed\
    \x02\x0b\x14\n\r\n\x05\x046\x02\x02\x03\x12\x04\xed\x02\x17\x18\n\x0c\n\
    \x04\x046\x02\x03\x12\x04\xee\x02\x04\x1a\n\x0f\n\x05\x046\x02\x03\x04\
    \x12\x06\xee\x02\x04\xed\x02\x19\n\r\n\x05\x046\x02\x03\x05\x12\x04\xee\
    \x02\x04\t\n\r\n\x05\x046\x02\x03\x01\x12\x04\xee\x02\n\x15\n\r\n\x05\
    \x046\x02\x03\x03\x12\x04\xee\x02\x18\x19\n\x0c\n\x02\x047\x12\x06\xf1\
    \x02\0\xf4\x02\x01\n\x0b\n\x03\x047\x01\x12\x04\xf1\x02\x08\x11\n\x0c\n\
    \x04\x047\x02\0\x12\x04\xf2\x02\x04\x18\n\x0f\n\x05\x047\x02\0\x04\x12\
    \x06\xf2\x02\x04\xf1\x02\x13\n\r\n\x05\x047\x02\0\x05\x12\x04\xf2\x02\
    \x04\n\n\r\n\x05\x047\x02\0\x01\x12\x04\xf2\x02\x0b\x13\n\r\n\x05\x047\
    \x02\0\x03\x12\x04\xf2\x02\x16\x17\n\x0c\n\x04\x047\x02\x01\x12\x04\xf3\
    \x02\x04\x14\n\x0f\n\x05\x047\x02\x01\x04\x12\x06\xf3\x02\x04\xf2\x02\
    \x18\n\r\n\x05\x047\x02\x01\x05\x12\x04\xf3\x02\x04\t\n\r\n\x05\x047\x02\
    \x01\x01\x12\x04\xf3\x02\n\x0f\n\r\n\x05\x047\x02\x01\x03\x12\x04\xf3\
    \x02\x12\x13\n\x0c\n\x02\x048\x12\x06\xf6\x02\0\xfb\x02\x01\n\x0b\n\x03\
    \x048\x01\x12\x04\xf6\x02\x08\x10\n\x0c\n\x04\x048\x02\0\x12\x04\xf7\x02\
    \x04\x10\n\x0f\n\x05\x048\x02\0\x04\x12\x06\xf7\x02\x04\xf6\x02\x12\n\r\
    \n\x05\x048\x02\0\x06\x12\x04\xf7\x02\x04\x06\n\r\n\x05\x048\x02\0\x01\
    \x12\x04\xf7\x02\x07\x0b\n\r\n\x05\x048\x02\0\x03\x12\x04\xf7\x02\x0e\
    \x0f\n\x0c\n\x04\x048\x02\x01\x12\x04\xf8\x02\x04\x18\n\x0f\n\x05\x048\
    \x02\x01\x04\x12\x06\xf8\x02\x04\xf7\x02\x10\n\r\n\x05\x048\x02\x01\x05\
    \x12\x04\xf8\x02\x04\n\n\r\n\x05\x048\x02\x01\x01\x12\x04\xf8\x02\x0b\
    \x13\n\r\n\x05\x048\x02\x01\x03\x12\x04\xf8\x02\x16\x17\n\x0c\n\x04\x048\
    \x02\x02\x12\x04\xf9\x02\x04\x16\n\x0f\n\x05\x048\x02\x02\x04\x12\x06\
    \xf9\x02\x04\xf8\x02\x18\n\r\n\x05\x048\x02\x02\x05\x12\x04\xf9\x02\x04\
    \t\n\r\n\x05\x048\x02\x02\x01\x12\x04\xf9\x02\n\x11\n\r\n\x05\x048\x02\
    \x02\x03\x12\x04\xf9\x02\x14\x15\n\x0c\n\x04\x048\x02\x03\x12\x04\xfa\
    \x02\x04\x1a\n\x0f\n\x05\x048\x02\x03\x04\x12\x06\xfa\x02\x04\xf9\x02\
    \x16\n\r\n\x05\x048\x02\x03\x05\x12\x04\xfa\x02\x04\t\n\r\n\x05\x048\x02\
    \x03\x01\x12\x04\xfa\x02\n\x15\n\r\n\x05\x048\x02\x03\x03\x12\x04\xfa\
    \x02\x18\x19\n\x0c\n\x02\x049\x12\x06\xfd\x02\0\x81\x03\x01\n\x0b\n\x03\
    \x049\x01\x12\x04\xfd\x02\x08\x10\n\x0c\n\x04\x049\x02\0\x12\x04\xfe\x02\
    \x04\x16\n\x0f\n\x05\x049\x02\0\x04\x12\x06\xfe\x02\x04\xfd\x02\x12\n\r\
    \n\x05\x049\x02\0\x06\x12\x04\xfe\x02\x04\x0c\n\r\n\x05\x049\x02\0\x01\
    \x12\x04\xfe\x02\r\x11\n\r\n\x05\x049\x02\0\x03\x12\x04\xfe\x02\x14\x15\
    \n\x0c\n\x04\x049\x02\x01\x12\x04\xff\x02\x04\"\n\r\n\x05\x049\x02\x01\
    \x04\x12\x04\xff\x02\x04\x0c\n\r\n\x05\x049\x02\x01\x06\x12\x04\xff\x02\
    \r\x16\n\r\n\x05\x049\x02\x01\x01\x12\x04\xff\x02\x17\x1d\n\r\n\x05\x049\
    \x02\x01\x03\x12\x04\xff\x02\x20!\n\x0c\n\x04\x049\x02\x02\x12\x04\x80\
    \x03\x04\"\n\r\n\x05\x049\x02\x02\x04\x12\x04\x80\x03\x04\x0c\n\r\n\x05\
    \x049\x02\x02\x06\x12\x04\x80\x03\r\x16\n\r\n\x05\x049\x02\x02\x01\x12\
    \x04\x80\x03\x17\x1d\n\r\n\x05\x049\x02\x02\x03\x12\x04\x80\x03\x20!\n\
    \x0c\n\x02\x04:\x12\x06\x83\x03\0\x86\x03\x01\n\x0b\n\x03\x04:\x01\x12\
    \x04\x83\x03\x08\x1b\n\x0c\n\x04\x04:\x02\0\x12\x04\x84\x03\x04\x18\n\
    \x0f\n\x05\x04:\x02\0\x04\x12\x06\x84\x03\x04\x83\x03\x1d\n\r\n\x05\x04:\
    \x02\0\x06\x12\x04\x84\x03\x04\x0b\n\r\n\x05\x04:\x02\0\x01\x12\x04\x84\
    \x03\x0c\x13\n\r\n\x05\x04:\x02\0\x03\x12\x04\x84\x03\x16\x17\n\x0c\n\
    \x04\x04:\x02\x01\x12\x04\x85\x03\x04\x12\n\x0f\n\x05\x04:\x02\x01\x04\
    \x12\x06\x85\x03\x04\x84\x03\x18\n\r\n\x05\x04:\x02\x01\x05\x12\x04\x85\
    \x03\x04\t\n\r\n\x05\x04:\x02\x01\x01\x12\x04\x85\x03\n\r\n\r\n\x05\x04:\
    \x02\x01\x03\x12\x04\x85\x03\x10\x11\n\x0c\n\x02\x04;\x12\x06\x88\x03\0\
    \x8c\x03\x01\n\x0b\n\x03\x04;\x01\x12\x04\x88\x03\x08\x1c\n\x0c\n\x04\
    \x04;\x02\0\x12\x04\x89\x03\x04#\n\x0f\n\x05\x04;\x02\0\x04\x12\x06\x89\
    \x03\x04\x88\x03\x1e\n\r\n\x05\x04;\x02\0\x06\x12\x04\x89\x03\x04\x11\n\
    \r\n\x05\x04;\x02\0\x01\x12\x04\x89\x03\x12\x1e\n\r\n\x05\x04;\x02\0\x03\
    \x12\x04\x89\x03!\"\n\x0c\n\x04\x04;\x02\x01\x12\x04\x8a\x03\x04\x15\n\
    \x0f\n\x05\x04;\x02\x01\x04\x12\x06\x8a\x03\x04\x89\x03#\n\r\n\x05\x04;\
    \x02\x01\x05\x12\x04\x8a\x03\x04\n\n\r\n\x05\x04;\x02\x01\x01\x12\x04\
    \x8a\x03\x0b\x10\n\r\n\x05\x04;\x02\x01\x03\x12\x04\x8a\x03\x13\x14\n\
    \x0c\n\x04\x04;\x02\x02\x12\x04\x8b\x03\x04\x16\n\x0f\n\x05\x04;\x02\x02\
    \x04\x12\x06\x8b\x03\x04\x8a\x03\x15\n\r\n\x05\x04;\x02\x02\x06\x12\x04\
    \x8b\x03\x04\x0c\n\r\n\x05\x04;\x02\x02\x01\x12\x04\x8b\x03\r\x11\n\r\n\
    \x05\x04;\x02\x02\x03\x12\x04\x8b\x03\x14\x15\n\x0c\n\x02\x04<\x12\x06\
    \x8e\x03\0\x91\x03\x01\n\x0b\n\x03\x04<\x01\x12\x04\x8e\x03\x08\x1f\n\
    \x0c\n\x04\x04<\x02\0\x12\x04\x8f\x03\x04\x18\n\x0f\n\x05\x04<\x02\0\x04\
    \x12\x06\x8f\x03\x04\x8e\x03!\n\r\n\x05\x04<\x02\0\x06\x12\x04\x8f\x03\
    \x04\x0b\n\r\n\x05\x04<\x02\0\x01\x12\x04\x8f\x03\x0c\x13\n\r\n\x05\x04<\
    \x02\0\x03\x12\x04\x8f\x03\x16\x17\n\x0c\n\x04\x04<\x02\x01\x12\x04\x90\
    \x03\x04\x18\n\x0f\n\x05\x04<\x02\x01\x04\x12\x06\x90\x03\x04\x8f\x03\
    \x18\n\r\n\x05\x04<\x02\x01\x05\x12\x04\x90\x03\x04\n\n\r\n\x05\x04<\x02\
    \x01\x01\x12\x04\x90\x03\x0b\x13\n\r\n\x05\x04<\x02\x01\x03\x12\x04\x90\
    \x03\x16\x17\n\x0c\n\x02\x04=\x12\x06\x93\x03\0\x98\x03\x01\n\x0b\n\x03\
    \x04=\x01\x12\x04\x93\x03\x08\x20\n\x0c\n\x04\x04=\x02\0\x12\x04\x94\x03\
    \x04#\n\x0f\n\x05\x04=\x02\0\x04\x12\x06\x94\x03\x04\x93\x03\"\n\r\n\x05\
    \x04=\x02\0\x06\x12\x04\x94\x03\x04\x11\n\r\n\x05\x04=\x02\0\x01\x12\x04\
    \x94\x03\x12\x1e\n\r\n\x05\x04=\x02\0\x03\x12\x04\x94\x03!\"\n\x0c\n\x04\
    \x04=\x02\x01\x12\x04\x95\x03\x04\x15\n\x0f\n\x05\x04=\x02\x01\x04\x12\
    \x06\x95\x03\x04\x94\x03#\n\r\n\x05\x04=\x02\x01\x05\x12\x04\x95\x03\x04\
    \n\n\r\n\x05\x04=\x02\x01\x01\x12\x04\x95\x03\x0b\x10\n\r\n\x05\x04=\x02\
    \x01\x03\x12\x04\x95\x03\x13\x14\n\x0c\n\x04\x04=\x02\x02\x12\x04\x96\
    \x03\x04\x12\n\x0f\n\x05\x04=\x02\x02\x04\x12\x06\x96\x03\x04\x95\x03\
    \x15\n\r\n\x05\x04=\x02\x02\x05\x12\x04\x96\x03\x04\t\n\r\n\x05\x04=\x02\
    \x02\x01\x12\x04\x96\x03\n\r\n\r\n\x05\x04=\x02\x02\x03\x12\x04\x96\x03\
    \x10\x11\n\x0c\n\x04\x04=\x02\x03\x12\x04\x97\x03\x04\x16\n\x0f\n\x05\
    \x04=\x02\x03\x04\x12\x06\x97\x03\x04\x96\x03\x12\n\r\n\x05\x04=\x02\x03\
    \x06\x12\x04\x97\x03\x04\x0c\n\r\n\x05\x04=\x02\x03\x01\x12\x04\x97\x03\
    \r\x11\n\r\n\x05\x04=\x02\x03\x03\x12\x04\x97\x03\x14\x15\n\x0c\n\x02\
    \x04>\x12\x06\x9a\x03\0\x9d\x03\x01\n\x0b\n\x03\x04>\x01\x12\x04\x9a\x03\
    \x08\x1a\n\x0c\n\x04\x04>\x02\0\x12\x04\x9b\x03\x04\x18\n\x0f\n\x05\x04>\
    \x02\0\x04\x12\x06\x9b\x03\x04\x9a\x03\x1c\n\r\n\x05\x04>\x02\0\x06\x12\
    \x04\x9b\x03\x04\x0b\n\r\n\x05\x04>\x02\0\x01\x12\x04\x9b\x03\x0c\x13\n\
    \r\n\x05\x04>\x02\0\x03\x12\x04\x9b\x03\x16\x17\n\x0c\n\x04\x04>\x02\x01\
    \x12\x04\x9c\x03\x04\x18\n\x0f\n\x05\x04>\x02\x01\x04\x12\x06\x9c\x03\
    \x04\x9b\x03\x18\n\r\n\x05\x04>\x02\x01\x05\x12\x04\x9c\x03\x04\t\n\r\n\
    \x05\x04>\x02\x01\x01\x12\x04\x9c\x03\n\x13\n\r\n\x05\x04>\x02\x01\x03\
    \x12\x04\x9c\x03\x16\x17\n\x0c\n\x02\x04?\x12\x06\x9f\x03\0\xa3\x03\x01\
    \n\x0b\n\x03\x04?\x01\x12\x04\x9f\x03\x08\x1b\n\x0c\n\x04\x04?\x02\0\x12\
    \x04\xa0\x03\x04#\n\x0f\n\x05\x04?\x02\0\x04\x12\x06\xa0\x03\x04\x9f\x03\
    \x1d\n\r\n\x05\x04?\x02\0\x06\x12\x04\xa0\x03\x04\x11\n\r\n\x05\x04?\x02\
    \0\x01\x12\x04\xa0\x03\x12\x1e\n\r\n\x05\x04?\x02\0\x03\x12\x04\xa0\x03!\
    \"\n\x0c\n\x04\x04?\x02\x01\x12\x04\xa1\x03\x04\x1c\n\x0f\n\x05\x04?\x02\
    \x01\x04\x12\x06\xa1\x03\x04\xa0\x03#\n\r\n\x05\x04?\x02\x01\x06\x12\x04\
    \xa1\x03\x04\x11\n\r\n\x05\x04?\x02\x01\x01\x12\x04\xa1\x03\x12\x16\n\r\
    \n\x05\x04?\x02\x01\x03\x12\x04\xa1\x03\x1a\x1b\n\x0c\n\x04\x04?\x02\x02\
    \x12\x04\xa2\x03\x04\x1c\n\x0f\n\x05\x04?\x02\x02\x04\x12\x06\xa2\x03\
    \x04\xa1\x03\x1c\n\r\n\x05\x04?\x02\x02\x06\x12\x04\xa2\x03\x04\x11\n\r\
    \n\x05\x04?\x02\x02\x01\x12\x04\xa2\x03\x12\x17\n\r\n\x05\x04?\x02\x02\
    \x03\x12\x04\xa2\x03\x1a\x1bb\x06proto3\
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
