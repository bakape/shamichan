mod clients;

use libc;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;

// Register a websocket client with a unique ID
#[no_mangle]
extern "C" fn ws_register_client(
    id: u64,
    ip: *const c_char,
    err: *mut *mut c_char,
) {
    if let Err(e) = clients::write(|c| -> Result<(), String> {
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
    }) {
        let size = e.len();
        unsafe {
            let buf = libc::malloc(size + 1) as *mut c_char;
            std::ptr::copy_nonoverlapping(
                CString::new(e).expect("error contains null bytes").as_ptr(),
                buf,
                size,
            );
            *buf.offset(size as isize) = 0;
            *err = buf;
        };
    }
}

// Remove client from registry
#[no_mangle]
extern "C" fn ws_unregister_client(id: u64) {
    clients::write(|c| c.remove(&id));
}
