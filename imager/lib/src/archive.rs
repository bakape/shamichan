use libflate::gzip;
use lzma::LzmaReader;
use std::io::Read;
use std::marker::Sized;
use std::slice;

// Detect if file is a TAR archive compressed with GZIP
#[no_mangle]
pub extern "C" fn is_tar_gz(b: *const u8, size: usize) -> bool {
	let buf = unsafe { slice::from_raw_parts(b, size) };
	if !buf.starts_with(b"\x1f\x8b\x08") {
		return false;
	}
	is_tar(gzip::Decoder::new(buf))
}

// Read the start of the file and determine, if it is a TAR archive
fn is_tar<D: Read + Sized, E>(decoder: Result<D, E>) -> bool {
	let r = match decoder {
		Ok(r) => r,
		_ => return false,
	};
	let mut decoded = Vec::with_capacity(262);
	match r.take(262).read_to_end(&mut decoded) {
		Ok(_) => decoded[257..].starts_with(b"ustar"),
		_ => false,
	}
}

// Detect if file is a TAR archive compressed with XZ
#[no_mangle]
pub extern "C" fn is_tar_xz(b: *const u8, size: usize) -> bool {
	let buf = unsafe { slice::from_raw_parts(b, size) };
	if !buf.starts_with(b"\xFD7zXZ\x00") {
		return false;
	}
	is_tar(LzmaReader::new_decompressor(buf))
}
