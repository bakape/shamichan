import { PostLink } from "./types";

// Types of IPC messages between tabs and the service worker
export const enum Type {
	// Add another thread to the watch list
	watchThread,
	// Remove a thread from the watched threads list
	unwatchThread,
};

// Basic interface for all messages
export interface Message {
	type: Type;
};

// Request to watch another thread
export interface WatchThreadRequest extends Message {
	thread: number;
	lastSeen: number;
};

// Unwatch a watched thread
export interface UnwatchThreadRequest extends Message {
	thread: number;
};

// Message received from the server's thread watcher service
export interface WatcherMessage {
	id: number;
	op: number;
	body: string;
	image?: string;
	links?: PostLink[];
	repliedToMe: boolean; // Set on the client side
};

// Notification about a new post appearing
export interface NewPostMessage extends Message {
	data: WatcherMessage;
};
