mod client;
mod clients;

use client::Client;
use libc;
use std::os::raw::c_char;
use std::rc::Rc;
use std::sync::Mutex;

// Wrapper for passing buffer references over the FFI
#[repr(C)]
pub struct WSBuffer {
    data: *const u8,
    size: usize,
}

impl AsRef<[u8]> for WSBuffer {
    fn as_ref(&self) -> &[u8] {
        unsafe { std::slice::from_raw_parts(self.data, self.size) }
    }
}

// Register a websocket client with a unique ID and return any error
#[no_mangle]
extern "C" fn ws_register_client(id: u64, ip: WSBuffer) -> *mut c_char {
    cast_error(clients::write(|c| {
        c.insert(
            id,
            Rc::new(Mutex::new(Client::new(
                id,
                std::str::from_utf8(ip.as_ref())
                    .map_err(|_| String::from("could not read IP string"))?
                    .parse()
                    .map_err(|err| format!("{}", err))?,
            ))),
        );
        Ok(())
    }))
}

// Cast result and allocate error message as owned C string, if any
pub fn cast_error<T>(r: Result<T, String>) -> *mut c_char {
    match r {
        Ok(_) => std::ptr::null_mut(),
        Err(err) => alloc_error(&err),
    }
}

// Allocate error message as owned C string, if any
fn alloc_error(err: &str) -> *mut c_char {
    let size = err.len();
    if size == 0 {
        std::ptr::null_mut()
    } else {
        unsafe {
            let buf = libc::malloc(size + 1) as *mut c_char;
            std::ptr::copy_nonoverlapping(
                err.as_ptr() as *const c_char,
                buf,
                size,
            );
            *buf.offset(size as isize) = 0;
            buf
        }
    }
}

// Send close message with optional error to client and unregister it
pub fn close_client(client_id: u64, err: &str) {
    // Go would still unregister the client eventually, but removing it early
    // will prevent any further message write attempts to it.
    clients::remove_client(client_id);

    unsafe { ws_close_client(client_id, alloc_error(err)) };
}

// Pass received message to Rust side. This operation never returns an error to
// simplify error propagation. All errors are propagated back to Go only using
// ws_close_client.
#[no_mangle]
extern "C" fn ws_receive_message(client_id: u64, msg: WSBuffer) {
    // Release lock on global collection as soon as possible.
    //
    // Client could be not found due to a race between the main client
    // goroutine and the reading goroutine.
    //
    // It's fine - unregistration can be eventual.
    if let Some(c) = clients::read(|cls| cls.get(&client_id).map(|c| c.clone()))
    {
        if let Err(err) = c.lock().unwrap().receive_message(msg.as_ref()) {
            close_client(client_id, &err.to_string());
        }
    }
}

// Remove client from registry
#[no_mangle]
extern "C" fn ws_unregister_client(id: u64) {
    clients::remove_client(id);
}

// Unref and potentially free a message on the Rust side
#[no_mangle]
extern "C" fn ws_unref_message(msg: *const WSBuffer) {
    unsafe { Rc::<WSBuffer>::from_raw(msg) }; // Drop it
}

// Linked from Go
extern "C" {
    fn ws_write_message(client_id: u64, msg: *const WSBuffer);
    fn ws_close_client(clientID: u64, err: *const c_char);
}
