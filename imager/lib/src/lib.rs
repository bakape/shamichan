extern crate libflate;
extern crate libc;

use std::io::Read;
use std::marker::Sized;
use libflate::gzip;
use libc::size_t;

// Detect if file is a TAR archive compressed with GZIP
#[no_mangle]
pub extern "C" fn is_tar_gz(b: *const u8, size: size_t) -> bool {
    let buf = unsafe { std::slice::from_raw_parts(b, size) };
    if !buf.starts_with(&[0x1f, 0x8b, 0x08]) {
        return false;
    }
    match gzip::Decoder::new(buf) {
        Ok(r) => is_tar(r),
        _ => false,
    }
}

fn is_tar<T: Read + Sized>(r: T) -> bool {
    let mut decoded = Vec::with_capacity(262);
    match r.take(262).read_to_end(&mut decoded) {
        Ok(_) => decoded[257..].starts_with("ustar".as_bytes()),
        _ => false,
    }
}
