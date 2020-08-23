use super::MessageType;
use bincode;
use flate2::write::{DeflateDecoder, DeflateEncoder};
use serde::{Deserialize, Serialize};
use std::io;
use std::io::Write;

/// Byte used for marking the start of a message
const HEADER: u8 = 174;

/// Byte used for escaping HEADER in massages
const ESCAPE: u8 = 255;

/// Appends 0 after HEADER byte to distinguish it from a message start
struct Escaper<W: Write> {
	w: W,
}

impl<W: Write> Escaper<W> {
	fn new(w: W) -> Escaper<W> {
		Self { w: w }
	}
}

impl<W: Write> Write for Escaper<W> {
	fn write(&mut self, mut buf: &[u8]) -> io::Result<usize> {
		let src_len = buf.len();

		while buf.len() > 0 {
			match buf.iter().position(|b| *b == HEADER) {
				Some(i) => {
					self.w.write_all(&buf[..i])?;
					self.w.write_all(&[HEADER, ESCAPE])?;
					buf = &buf[i + 1..];
				}
				None => {
					self.w.write_all(&buf)?;
					return Ok(src_len);
				}
			}
		}

		Ok(src_len)
	}

	fn flush(&mut self) -> io::Result<()> {
		self.w.flush()
	}
}

/// Streaming message set encoder
#[derive(Debug)]
pub struct Encoder {
	w: DeflateEncoder<Vec<u8>>,
}

impl Default for Encoder {
	fn default() -> Self {
		Self::new(Default::default())
	}
}

impl Encoder {
	/// Create new encoder for building message streams, which will have its
	/// output written to the passed output stream.
	pub fn new(mut w: Vec<u8>) -> Self {
		Self::init_single_message(&mut w);
		Self {
			w: DeflateEncoder::new(w, flate2::Compression::default()),
		}
	}

	/// Utility for only encoding a single message without any batching
	pub fn encode(
		typ: MessageType,
		payload: &impl Serialize,
	) -> io::Result<Vec<u8>> {
		let mut enc = Encoder::new(Vec::new());
		enc.write_message(typ, payload)?;
		enc.finish()
	}

	/// Indicate this is single message and not a concatenated vector of
	/// messages
	fn init_single_message(w: &mut Vec<u8>) {
		w.push(0);
	}

	/// Join already encoded messages into a single stream
	pub fn join<I, A>(encoded: I) -> Vec<u8>
	where
		I: AsRef<[A]>,
		A: AsRef<[u8]>,
	{
		let enc = encoded.as_ref();
		let mut w = Vec::with_capacity(
			enc.iter().map(|b| b.as_ref().len() + 4).sum::<usize>() + 1,
		);

		// Indicates this is a concatenated vector of messages
		w.push(1);

		for msg in enc.iter() {
			let s = msg.as_ref();
			w.extend((s.len() as i32).to_le_bytes().iter());
			w.extend(s.iter());
		}

		w
	}

	/// Flush any pending data to output stream
	pub fn flush(&mut self) -> io::Result<()> {
		self.w.flush()
	}

	/// Write message to underlying writer
	pub fn write_message(
		&mut self,
		t: MessageType,
		payload: &impl Serialize,
	) -> io::Result<()> {
		self.w.write_all(&[HEADER, t as u8])?;
		bincode::serialize_into(&mut Escaper::new(&mut self.w), payload)
			.map_err(|err| {
				io::Error::new(io::ErrorKind::Other, err.to_string())
			})
	}

	/// Consumes this encoder, flushing the output stream and returning the
	/// underlying writer
	pub fn finish(self) -> io::Result<Vec<u8>> {
		self.w.finish()
	}

	/// Resets the state of this encoder entirely, swapping out the output
	/// stream for another.
	///
	/// This function will finish encoding the current stream into the current
	/// output stream before swapping out the two output streams.
	pub fn reset(&mut self, mut w: Vec<u8>) -> io::Result<Vec<u8>> {
		Self::init_single_message(&mut w);
		self.w.reset(w)
	}
}

/// Decompresses and decodes message batch.
#[derive(Debug)]
pub struct Decoder {
	splitter: MessageSplitter,
	off: usize,
}

impl Decoder {
	/// Create new decoder for reading the passed buffer
	pub fn new(r: &[u8]) -> Result<Self, io::Error> {
		Ok(Self {
			splitter: Self::fill_splitter(MessageSplitter::new(), r)?,
			off: 0,
		})
	}

	/// Decode buffer into an existing message splitter and return it on success
	fn fill_splitter(
		mut dst: MessageSplitter,
		mut r: &[u8],
	) -> Result<MessageSplitter, io::Error> {
		use std::io::{Error, ErrorKind};

		macro_rules! error {
			($kind:ident, $msg:expr) => {
				return Err(Error::new(ErrorKind::$kind, $msg));
			};
		}

		if r.len() == 0 {
			error!(UnexpectedEof, "zero length buffer");
		}

		match r[0] {
			0 => {
				// Single compressed message
				let mut dd = DeflateDecoder::new(dst);
				dd.write_all(&r[1..])?;
				Ok(dd.finish()?)
			}
			1 => {
				// Vector of compressed messages
				r = &r[1..];
				while r.len() > 0 {
					#[rustfmt::skip]
					macro_rules! check_len {
						($n:expr) => {
							if r.len() < $n {
								error!(
									InvalidData,
									format!(
										concat!(
											"incomplete message in vector: ",
											"min_length={} msg={:?}"
										),
										$n, r,
									)
								);
							}
						};
					}

					check_len!(4);
					let len = i32::from_le_bytes({
						let mut arr: [u8; 4] = unsafe {
							std::mem::MaybeUninit::uninit().assume_init()
						};
						arr.copy_from_slice(&r[..4]);
						arr
					}) as usize;
					r = &r[4..];

					check_len!(len);
					dst = Self::fill_splitter(dst, &r[..len])?;
					r = &r[len..];
				}
				Ok(dst)
			}
			_ => error!(InvalidData, format!("invalid header byte: {}", r[0])),
		}
	}

	/// Return next message type, if any.
	/// Returns None, if entire message stream has been consumed.
	///
	/// This method does not advance the decoder. Either read_next() or
	/// skip_next() need to be called to advance it.
	pub fn peek_type(&mut self) -> Option<MessageType> {
		self.splitter.message_types.get(self.off).copied()
	}

	/// Skip reading next message and advance the decoder
	pub fn skip_next(&mut self) {
		self.off += 1;
	}

	/// Read, decode and return next message payload from stream
	pub fn read_next<'a, 's: 'a, T: Deserialize<'a>>(
		&'s mut self,
	) -> io::Result<T> {
		match self.splitter.message_starts.get(self.off) {
			None => Err(io::Error::from(io::ErrorKind::NotFound)),
			Some(i) => {
				let to = *self
					.splitter
					.message_starts
					.get(self.off + 1)
					.unwrap_or(&self.splitter.buf.len());
				let buf = &self.splitter.buf[*i..to];
				let res = bincode::deserialize(buf).map_err(|err| {
					io::Error::new(io::ErrorKind::InvalidData, err.to_string())
				})?;
				self.off += 1;
				Ok(res)
			}
		}
	}
}

/// Splits and unescapes messages
#[derive(Debug)]
struct MessageSplitter {
	buf: Vec<u8>,
	message_starts: Vec<usize>,
	message_types: Vec<MessageType>,
}

impl MessageSplitter {
	fn new() -> Self {
		Self {
			buf: Vec::with_capacity(1 << 10),
			message_starts: Vec::new(),
			message_types: Vec::new(),
		}
	}
}

impl Write for MessageSplitter {
	fn write(&mut self, mut buf: &[u8]) -> io::Result<usize> {
		fn err_invalid_data<T>(msg: impl Into<String>) -> io::Result<T> {
			Err(io::Error::new(io::ErrorKind::InvalidData, msg.into()))
		}

		let src_len = buf.len();

		while buf.len() > 0 {
			match buf.iter().position(|b| *b == HEADER) {
				Some(i) => {
					if i == buf.len() - 1 {
						return err_invalid_data(
							"buffer ends with header byte",
						);
					}

					let next = buf[i + 1];
					if next != ESCAPE {
						// Found next message
						match num::FromPrimitive::from_u8(next) {
							Some(typ) => {
								self.message_starts.push(self.buf.len() + i);
								self.message_types.push(typ);
								self.buf.extend(&buf[..i]);
							}
							None => {
								return err_invalid_data(format!(
									"invalid message type: {}",
									buf[1],
								))
							}
						};
					} else {
						// Escaped byte
						self.buf.extend(&buf[..i + 1]);
					}
					buf = &buf[i + 2..];
				}
				None => {
					self.buf.extend(buf);
					return Ok(src_len);
				}
			}
		}

		Ok(src_len)
	}

	fn flush(&mut self) -> io::Result<()> {
		Ok(())
	}
}

#[cfg(test)]
mod tests {
	use serde::{Deserialize, Serialize};

	type Result<T = ()> = std::result::Result<T, Box<dyn std::error::Error>>;

	#[derive(Serialize, Deserialize, Debug, PartialEq, Eq)]
	struct SimpleMessage<'a> {
		a: u64,
		b: &'a str,
		buf: &'a [u8],
	}

	const WITH_HEADER: [u8; 6] = [1, 2, 3, super::HEADER, 1, 3];

	#[test]
	fn simple_message_stream() -> Result {
		assert_decoded(&gen_message(0, 3)?)
	}

	#[test]
	fn encoded_message_vector() -> Result {
		assert_decoded(&super::Encoder::join(&[
			gen_message(0, 1)?,
			gen_message(2, 3)?,
		]))
	}

	fn gen_message(from: u64, to: u64) -> Result<Vec<u8>> {
		let mut enc = super::Encoder::new(Vec::<u8>::new());
		for i in from..=to {
			enc.write_message(
				num::FromPrimitive::from_u64(i).unwrap(),
				&SimpleMessage {
					a: i,
					b: &i.to_string(),
					// Include HEADER for escape testing
					buf: &WITH_HEADER,
				},
			)?;
		}

		Ok(enc.finish()?)
	}

	fn assert_decoded(buf: &[u8]) -> Result {
		let mut dec = super::Decoder::new(buf)?;
		for i in 0..=3 {
			assert_eq!(dec.peek_type(), num::FromPrimitive::from_u64(i));
			let res: SimpleMessage = dec.read_next()?;
			assert_eq!(
				res,
				SimpleMessage {
					a: i,
					b: &i.to_string(),
					// Include HEADER for escape testing
					buf: &WITH_HEADER,
				},
			);
		}
		assert_eq!(dec.peek_type(), None);
		assert_eq!(
			(dec.read_next() as std::io::Result<u64>).map_err(|e| e.kind()),
			Err(std::io::ErrorKind::NotFound)
		);

		Ok(())
	}
}
