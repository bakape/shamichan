use crate::util;
use serde::{Deserialize, Serialize};
use wasm_bindgen::JsCast;

/// Key used to store authentication key pair in local storage
const LOCAL_STORAGE_KEY: &str = "key_pair";

/// Key pair used to authenticate with server
#[derive(Serialize, Deserialize, Default, Clone, Eq, PartialEq)]
pub struct KeyPair {
	/// Private key
	pub private: Vec<u8>,

	/// Public key
	pub public: Vec<u8>,

	/// ID the key is registered to on the server
	pub id: Option<uuid::Uuid>,
}

impl KeyPair {
	/// Store in local storage
	#[cold]
	pub fn store(&self) -> util::Result {
		let mut dst = Vec::with_capacity(1 << 10);
		{
			// Block causes drop of encoders and thus releases dst reference
			let mut b64_w =
				base64::write::EncoderWriter::new(&mut dst, base64::STANDARD);
			let mut w = flate2::write::DeflateEncoder::new(
				&mut b64_w,
				flate2::Compression::default(),
			);
			bincode::serialize_into(&mut w, self)?;
			w.finish()?;
			b64_w.finish()?;
		}

		util::local_storage()
			.set_item(LOCAL_STORAGE_KEY, &String::from_utf8(dst)?)?;
		Ok(())
	}

	/// Load from local storage or generate a new one
	#[cold]
	pub async fn load() -> util::Result<KeyPair> {
		Ok(match util::local_storage().get_item(LOCAL_STORAGE_KEY)? {
			Some(s) => {
				bincode::deserialize_from(flate2::read::DeflateDecoder::new(
					base64::read::DecoderReader::new(
						&mut s.as_bytes(),
						base64::STANDARD,
					),
				))?
			}
			None => {
				let kp = Self::generate().await?;
				kp.store()?;
				kp
			}
		})
	}

	fn crypto() -> util::Result<web_sys::SubtleCrypto> {
		Ok(util::window().crypto()?.subtle())
	}

	/// Return dict describing the key pair algorithm
	fn algo_dict() -> util::Result<js_sys::Object> {
		let algo = js_sys::Object::new();

		#[rustfmt::skip]
		macro_rules! set {
			($k:expr, $v:expr) => {
				js_sys::Reflect::set(
					&algo,
					&$k.into(),
					&$v.into(),
				)?;
			};
		}

		set!("name", "RSASSA-PKCS1-v1_5");
		set!("modulusLength", 4096);
		set!(
			"publicExponent",
			js_sys::Uint8Array::new(
				&util::into_js_array(
					[1_u8, 0, 1].iter().map(|n| js_sys::Number::from(*n))
				)
				.into()
			)
		);
		set!("hash", "SHA-256");

		Ok(algo)
	}

	/// Return key usage array to pass to JS
	fn usages() -> wasm_bindgen::JsValue {
		util::into_js_array(Some("sign")).into()
	}

	/// Generate a new key pair
	#[cold]
	async fn generate() -> util::Result<KeyPair> {
		let pair = wasm_bindgen_futures::JsFuture::from(
			Self::crypto()?.generate_key_with_object(
				&Self::algo_dict()?,
				true,
				&Self::usages(),
			)?,
		)
		.await?
		.dyn_into::<js_sys::Object>()?;

		async fn get_vec(
			pair: &js_sys::Object,
			prop: &str,
			format: &str,
		) -> util::Result<Vec<u8>> {
			Ok(js_sys::Uint8Array::new(
				&wasm_bindgen_futures::JsFuture::from(
					KeyPair::crypto()?.export_key(
						format,
						&js_sys::Reflect::get(&pair, &prop.into())?
							.dyn_into::<web_sys::CryptoKey>()?,
					)?,
				)
				.await?
				.into(),
			)
			.to_vec())
		}

		let (priv_key, pub_key) = futures::future::join(
			get_vec(&pair, "privateKey", "pkcs8"),
			get_vec(&pair, "publicKey", "spki"),
		)
		.await;
		Ok(KeyPair {
			private: priv_key?,
			public: pub_key?,
			id: None,
		})
	}

	/// Sign SHA-256 digest of passed buffer
	pub async fn sign(
		&self,
		buf: &mut [u8],
	) -> util::Result<common::payloads::Signature> {
		use js_sys::Uint8Array;
		use wasm_bindgen_futures::JsFuture;

		let crypto = Self::crypto()?;
		let mut arr: [u8; 512] =
			unsafe { std::mem::MaybeUninit::uninit().assume_init() };
		let js_arr = Uint8Array::new(
			&JsFuture::from(crypto.sign_with_str_and_u8_array(
				"RSASSA-PKCS1-v1_5",
				{
					&JsFuture::from(
						crypto.import_key_with_object(
							"pkcs8",
							&Uint8Array::new(
								&util::into_js_array(
									self.private.iter().copied(),
								)
								.into(),
							)
							.into(),
							&Self::algo_dict()?,
							true,
							&Self::usages(),
						)?,
					)
					.await?
					.dyn_into::<web_sys::CryptoKey>()?
				},
				buf,
			)?)
			.await?
			.into(),
		);
		if js_arr.length() != 512 {
			Err(format!("unexpected signature length: {}", js_arr.length()))?;
		}
		js_arr.copy_to(&mut arr);
		Ok(common::payloads::Signature(arr))
	}
}
