import options from "."
import { page } from "../state"
import { setAttrs } from "../util"
import { handlers, message, connSM, connState, send } from "../connection"
import { getInvidiousData } from "../posts/embed"
import { truncateString } from "../util"
import { secondsToTimeExact } from "../util/time"
declare var YT: any

type Msg = {
	cmd: string
	data: any
}

type DataPlaylist = {
	playlist: Video[]
	currentTime: number
}

type DataPush = {
	video: Video
}

type DataSyncTime = {
	currentTime: number
}

type Video = {
	url: string
	title: string
	duration: number
	type: string
}

let playlist = [] as Video[]
let currentTime: number
let timeSyncCallback: (currentTime: number) => void = null

const cinemaDiv = document.getElementById("cinema-panel"),
		playerDiv = document.getElementById("cinema-player"),
		playlistDiv = document.getElementById("cinema-playlist"),
		playlistEmptyDiv = document.getElementById("cinema-playlist-empty")
let ytAPI: HTMLElement = null

// update initial state of containers
// should be called only on connection start
function render() {
	show()
	updateErrors()
	updateVideo()
	updatePlaylist()
}

// rewrites playlist entires based on playlist array
function updatePlaylist() {
	let playlistOl = document.getElementById("cinema-playlist-entries")
	if (playlistOl) {
		playlistOl.remove()
	}
	if (playlist.length === 0) {
		return
	}
	playlistOl = document.createElement("ol")
	playlistOl.setAttribute("id", "cinema-playlist-entries")
	playlistDiv.append(playlistOl)
	for (let i = 0; i < playlist.length; i++) {
		const el = document.createElement("li")
		el.setAttribute("class", "cinema-playlist-entry")
		el.innerHTML = `<a href="${playlist[i].url}" title="${playlist[i].title}">` +
			`${truncateString(playlist[i].title, 80)}</a>` +
			`<span id="cinema-video-time">${secondsToTimeExact(playlist[i].duration/1000)}</span>`
		playlistOl.append(el)
	}
}

// functions appending element with video and creating callback for synchronizing time
const embedders: { [key: string]: (url: string) => any } = {
	"invidious": embedInvidious,
	"raw": embedRaw,
	"youtube": embedYoutube,
}

async function embedInvidious(url: string): Promise<void> {
	const data = await getInvidiousData(new URL(url))
	const vidEl = document.createElement("video") as HTMLVideoElement
	setAttrs(vidEl, {
		id: "cinema-video",
		src: data.formatStreams[0].url,
		autoplay: "true",
		controls: "true",
	})
	playerDiv.append(vidEl)
	timeSyncCallback = (currentTime: number) => {
		const vidEl = document.getElementById("cinema-video") as HTMLVideoElement
		if (Math.abs(vidEl.currentTime-currentTime) > 1) {
			vidEl.currentTime = currentTime
		}
	}
	syncVideo()
	vidEl.volume = options.audioVolume / 100
}

function embedRaw(url: string) {
	const vidEl = document.createElement("video") as HTMLVideoElement
	setAttrs(vidEl, {
		id: "cinema-video",
		src: url,
		autoplay: "true",
		controls: "true",
	})
	playerDiv.append(vidEl)
	timeSyncCallback = (currentTime: number) => {
		const vidEl = document.getElementById("cinema-video") as HTMLVideoElement
		if (Math.abs(vidEl.currentTime-currentTime) > 1) {
			vidEl.currentTime = currentTime
		}
	}
	syncVideo()
	vidEl.volume = options.audioVolume / 100
}

function embedYoutube(url: string) {
	const vidEl = document.createElement("div")
	vidEl.setAttribute("id", "cinema-video")
	playerDiv.append(vidEl)

	let player: any
	function initPlayer() {
		player = new YT.Player("cinema-video", {
			videoId: new URL(url).searchParams.get("v"),
			playerVars: {
				"autoplay": 1,
				"iv_load_policy": 3
			},
			events: {
				onReady: onPlayerReady
			}
		})
	}
	function onPlayerReady() {
		player.setVolume(options.audioVolume)
		timeSyncCallback = (currentTime: number) => {
			if (Math.abs(player.getCurrentTime()-currentTime) > 1) {
				player.seekTo(currentTime)
			}
		}
		syncVideo()
	}
	if (ytAPI) {
		initPlayer()
	} else {
		ytAPI = document.createElement("script")
		setAttrs(ytAPI, {
			src: "https://www.youtube.com/iframe_api",
			type: "text/javascript"
		})
		document.getElementsByTagName("head")[0].appendChild(ytAPI)
		function onAPIReadyCallback() {
			initPlayer()
		}
		(window as any).onYouTubeIframeAPIReady = onAPIReadyCallback
	}
}

// deletes current video and embeds next from playlist if there is any using embedders
function updateVideo() {
	let vidEl = document.getElementById("cinema-video")
	if (vidEl) {
		vidEl.remove()
	}
	timeSyncCallback = null
	if (playlist.length == 0) {
		return
	}
	const vid = playlist[0] as Video
	embedders[vid.type](vid.url)
}

// if playing video is present sets its time to server-synced value
function syncVideo() {
	if (!timeSyncCallback) {
		return
	}
	timeSyncCallback(currentTime)
}

// display or undsiplay errors based on their presence
function updateErrors() {
	if (playlist.length > 0) {
		playlistEmptyDiv.hidden = true
	} else {
		playlistEmptyDiv.hidden = false
	}
}

function messageHandler(msg: Msg) {
	switch(msg.cmd) {
		case "playlist": {
			const data = (msg.data as DataPlaylist)
			if (data.playlist) {
				playlist = data.playlist
			} else {
				playlist = []
			}
			currentTime = data.currentTime/1000
			render()
			break
		}
		case "push": {
			const video = (msg.data as DataPush).video
			playlist.push(video)
			if (playlist.length == 1) {
				currentTime = 0
				updateVideo()
			}
			updatePlaylist()
			updateErrors()
			break
		}
		case "pop": {
			playlist.shift()
			currentTime = 0
			updateVideo()
			updatePlaylist()
			updateErrors()
			break
		}
		case "syncTime": {
			currentTime = (msg.data as DataSyncTime).currentTime/1000
			syncVideo()
			break
		}
	}
}

function subscribe() {
	if (!page.thread) return
	send(message.cinemaSubscription, null)
}

function unsubscribe() {
	hide()
	let vidEl = document.getElementById("cinema-video")
	if (vidEl) {
		vidEl.remove()
	}
	send(message.cinemaCancelSubscription, null);
}

function show() {
	cinemaDiv.setAttribute("style", "display: block;")
}

function hide() {
	cinemaDiv.setAttribute("style", "display: none;")
}

export function initCinema() {
	handlers[message.cinemaSubscription] =  messageHandler;
	connSM.on(connState.synced, () => {
		if (options.cinema && !options.workModeToggle) subscribe()
	})
	options.onChange("cinema", on => {
		if (!options.workModeToggle) on ? subscribe() : unsubscribe()
	})
	options.onChange("workModeToggle", on => {
		if (options.cinema) on ? unsubscribe() : subscribe()
	})
}
