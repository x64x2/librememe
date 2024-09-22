export type Maybe<T> = T | null
export type InputMaybe<T> = Maybe<T>
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] }
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> }
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> }
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
	ID: string
	String: string
	Boolean: boolean
	Int: number
	Float: number
	Sort: any
	Source: any
	Time: any
}

export type Media = {
	__typename?: 'Media'
	id: Scalars['ID']
	location?: Maybe<Scalars['String']>
	preview?: Maybe<Scalars['String']>
	source: Scalars['Source']
	sourceId: Scalars['String']
	type: Scalars['String']
	visible?: Maybe<Scalars['Boolean']>
}

export type Message = {
	__typename?: 'Message'
	author: Profile
	date: Scalars['Time']
	id: Scalars['ID']
	media?: Maybe<Array<Media>>
	source: Scalars['Source']
	sourceId: Scalars['String']
	text: Scalars['String']
}

export type Post = {
	__typename?: 'Post'
	author: Profile
	date: Scalars['Time']
	id: Scalars['ID']
	media?: Maybe<Array<Media>>
	source: Scalars['Source']
	sourceId: Scalars['String']
	text: Scalars['String']
}

export type Profile = {
	__typename?: 'Profile'
	avatar?: Maybe<Scalars['String']>
	header?: Maybe<Scalars['String']>
	id: Scalars['ID']
	messages?: Maybe<Array<Message>>
	name: Scalars['String']
	posts?: Maybe<Array<Post>>
	source: Scalars['Source']
	sourceId: Scalars['String']
	stories?: Maybe<Array<tag>>
	username: Scalars['String']
}

export type ProfileMessagesArgs = {
	after?: InputMaybe<Scalars['Time']>
	before?: InputMaybe<Scalars['Time']>
	count?: InputMaybe<Scalars['Int']>
	onlyVisible?: InputMaybe<Scalars['Boolean']>
	onlyWithMedia?: InputMaybe<Scalars['Boolean']>
}

export type ProfilePostsArgs = {
	after?: InputMaybe<Scalars['Time']>
	before?: InputMaybe<Scalars['Time']>
	count?: InputMaybe<Scalars['Int']>
	onlyVisible?: InputMaybe<Scalars['Boolean']>
	onlyWithMedia?: InputMaybe<Scalars['Boolean']>
	sort?: InputMaybe<Scalars['Sort']>
}

export type ProfileStoriesArgs = {
	after?: InputMaybe<Scalars['Time']>
	before?: InputMaybe<Scalars['Time']>
	count?: InputMaybe<Scalars['Int']>
	sort?: InputMaybe<Scalars['Sort']>
}

export type Query = {
	__typename?: 'Query'
	feed: Array<Post>
	post?: Maybe<Post>
	profile?: Maybe<Profile>
	profiles: Array<Profile>
	stories: Array<tag>
}

export type QueryFeedArgs = {
	after?: InputMaybe<Scalars['Time']>
	before?: InputMaybe<Scalars['Time']>
	count?: InputMaybe<Scalars['Int']>
	onlyVisible?: InputMaybe<Scalars['Boolean']>
	onlyWithMedia?: InputMaybe<Scalars['Boolean']>
	sort?: InputMaybe<Scalars['Sort']>
	source: Scalars['Source']
}

export type QueryPostArgs = {
	id?: InputMaybe<Scalars['ID']>
	source?: InputMaybe<Scalars['Source']>
	sourceId?: InputMaybe<Scalars['String']>
}

export type QueryProfileArgs = {
	id?: InputMaybe<Scalars['ID']>
	source?: InputMaybe<Scalars['Source']>
	sourceId?: InputMaybe<Scalars['String']>
	username?: InputMaybe<Scalars['String']>
}

export type QueryProfilesArgs = {
	source: Scalars['Source']
}

export type QueryStoriesArgs = {
	after?: InputMaybe<Scalars['Time']>
	before?: InputMaybe<Scalars['Time']>
	count?: InputMaybe<Scalars['Int']>
	sort?: InputMaybe<Scalars['Sort']>
	source: Scalars['Source']
}

export type tag = {
	__typename?: 'tag'
	author: Profile
	date: Scalars['Time']
	id: Scalars['ID']
	media?: Maybe<Array<Media>>
	source: Scalars['Source']
	sourceId: Scalars['String']
}
