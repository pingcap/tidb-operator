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

const METHOD_IMPORT_KV_OPEN: ::grpcio::Method<super::import_kvpb::OpenRequest, super::import_kvpb::OpenResponse> = ::grpcio::Method {
    ty: ::grpcio::MethodType::Unary,
    name: "/import_kvpb.ImportKV/Open",
    req_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
    resp_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
};

const METHOD_IMPORT_KV_WRITE: ::grpcio::Method<super::import_kvpb::WriteRequest, super::import_kvpb::WriteResponse> = ::grpcio::Method {
    ty: ::grpcio::MethodType::ClientStreaming,
    name: "/import_kvpb.ImportKV/Write",
    req_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
    resp_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
};

const METHOD_IMPORT_KV_CLOSE: ::grpcio::Method<super::import_kvpb::CloseRequest, super::import_kvpb::CloseResponse> = ::grpcio::Method {
    ty: ::grpcio::MethodType::Unary,
    name: "/import_kvpb.ImportKV/Close",
    req_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
    resp_mar: ::grpcio::Marshaller { ser: ::grpcio::pb_ser, de: ::grpcio::pb_de },
};

pub struct ImportKvClient {
    client: ::grpcio::Client,
}

impl ImportKvClient {
    pub fn new(channel: ::grpcio::Channel) -> Self {
        ImportKvClient {
            client: ::grpcio::Client::new(channel),
        }
    }

    pub fn open_opt(&self, req: &super::import_kvpb::OpenRequest, opt: ::grpcio::CallOption) -> ::grpcio::Result<super::import_kvpb::OpenResponse> {
        self.client.unary_call(&METHOD_IMPORT_KV_OPEN, req, opt)
    }

    pub fn open(&self, req: &super::import_kvpb::OpenRequest) -> ::grpcio::Result<super::import_kvpb::OpenResponse> {
        self.open_opt(req, ::grpcio::CallOption::default())
    }

    pub fn open_async_opt(&self, req: &super::import_kvpb::OpenRequest, opt: ::grpcio::CallOption) -> ::grpcio::Result<::grpcio::ClientUnaryReceiver<super::import_kvpb::OpenResponse>> {
        self.client.unary_call_async(&METHOD_IMPORT_KV_OPEN, req, opt)
    }

    pub fn open_async(&self, req: &super::import_kvpb::OpenRequest) -> ::grpcio::Result<::grpcio::ClientUnaryReceiver<super::import_kvpb::OpenResponse>> {
        self.open_async_opt(req, ::grpcio::CallOption::default())
    }

    pub fn write_opt(&self, opt: ::grpcio::CallOption) -> ::grpcio::Result<(::grpcio::ClientCStreamSender<super::import_kvpb::WriteRequest>, ::grpcio::ClientCStreamReceiver<super::import_kvpb::WriteResponse>)> {
        self.client.client_streaming(&METHOD_IMPORT_KV_WRITE, opt)
    }

    pub fn write(&self) -> ::grpcio::Result<(::grpcio::ClientCStreamSender<super::import_kvpb::WriteRequest>, ::grpcio::ClientCStreamReceiver<super::import_kvpb::WriteResponse>)> {
        self.write_opt(::grpcio::CallOption::default())
    }

    pub fn close_opt(&self, req: &super::import_kvpb::CloseRequest, opt: ::grpcio::CallOption) -> ::grpcio::Result<super::import_kvpb::CloseResponse> {
        self.client.unary_call(&METHOD_IMPORT_KV_CLOSE, req, opt)
    }

    pub fn close(&self, req: &super::import_kvpb::CloseRequest) -> ::grpcio::Result<super::import_kvpb::CloseResponse> {
        self.close_opt(req, ::grpcio::CallOption::default())
    }

    pub fn close_async_opt(&self, req: &super::import_kvpb::CloseRequest, opt: ::grpcio::CallOption) -> ::grpcio::Result<::grpcio::ClientUnaryReceiver<super::import_kvpb::CloseResponse>> {
        self.client.unary_call_async(&METHOD_IMPORT_KV_CLOSE, req, opt)
    }

    pub fn close_async(&self, req: &super::import_kvpb::CloseRequest) -> ::grpcio::Result<::grpcio::ClientUnaryReceiver<super::import_kvpb::CloseResponse>> {
        self.close_async_opt(req, ::grpcio::CallOption::default())
    }
    pub fn spawn<F>(&self, f: F) where F: ::futures::Future<Item = (), Error = ()> + Send + 'static {
        self.client.spawn(f)
    }
}

pub trait ImportKv {
    fn open(&self, ctx: ::grpcio::RpcContext, req: super::import_kvpb::OpenRequest, sink: ::grpcio::UnarySink<super::import_kvpb::OpenResponse>);
    fn write(&self, ctx: ::grpcio::RpcContext, stream: ::grpcio::RequestStream<super::import_kvpb::WriteRequest>, sink: ::grpcio::ClientStreamingSink<super::import_kvpb::WriteResponse>);
    fn close(&self, ctx: ::grpcio::RpcContext, req: super::import_kvpb::CloseRequest, sink: ::grpcio::UnarySink<super::import_kvpb::CloseResponse>);
}

pub fn create_import_kv<S: ImportKv + Send + Clone + 'static>(s: S) -> ::grpcio::Service {
    let mut builder = ::grpcio::ServiceBuilder::new();
    let instance = s.clone();
    builder = builder.add_unary_handler(&METHOD_IMPORT_KV_OPEN, move |ctx, req, resp| {
        instance.open(ctx, req, resp)
    });
    let instance = s.clone();
    builder = builder.add_client_streaming_handler(&METHOD_IMPORT_KV_WRITE, move |ctx, req, resp| {
        instance.write(ctx, req, resp)
    });
    let instance = s.clone();
    builder = builder.add_unary_handler(&METHOD_IMPORT_KV_CLOSE, move |ctx, req, resp| {
        instance.close(ctx, req, resp)
    });
    builder.build()
}
