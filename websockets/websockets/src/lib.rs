mod clients;

use libc;
use std::ffi::CStr;
use std::os::raw::c_char;
use std::rc::Rc;

// Websocket message passed to Go as an Rc-associated pointer
#[repr(C)]
pub struct WSMessage {
    data: *const u8,
    size: usize,
}

// Register a websocket client with a unique ID and return any error
#[no_mangle]
extern "C" fn ws_register_client(id: u64, ip: *const c_char) -> *mut c_char {
    cast_error(clients::write(|c| {
        c.insert(
            id,
            clients::Client {
                id: id,
                ip: unsafe { CStr::from_ptr(ip) }
                    .to_str()
                    .map_err(|_| String::from("could not read IP string"))?
                    .parse()
                    .map_err(|err| format!("{}", err))?,
            },
        );
        Ok(())
    }))
}

// Cast result and Allocate error message as owned C string, if any
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

// Remove client from registry
#[no_mangle]
extern "C" fn ws_unregister_client(id: u64) {
    clients::remove_client(id);
}

// Unref and potentially free a message on the Rust side
#[no_mangle]
extern "C" fn ws_unref_message(msg: *const WSMessage) {
    unsafe { Rc::<WSMessage>::from_raw(msg) }; // Drop it
}

// Linked from Go
extern "C" {
    fn ws_write_message(client_id: u64, msg: *const WSMessage);
    fn ws_close_client(clientID: u64, err: *const c_char);
}
