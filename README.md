```
[+] async pubgm

A python wrapper for the PUBGM developer API

async version of https://github.com/ramonsaraiva/pubg-python
check his repo for the rest of the doc

[-] install
*sync client is stil available in master branch
*get the async version from async branch

pip install git+https://github.com/x64x2/asynpubgm

[-] api changes
*await obj.get() obj.fetch() or obj[x] calls
*you can not longer do "for x in QuerySet", use "for x in await QuerySet.all()"

async def ent():

	pubg = pubg_python.AioPUBG("api_key", shard=pubg_python.Shard.STEAM)

	player = await pubg.players().filter(player_names=["auteuil"])[0]
	player = await pubg.players().filter(player_names=["auteuil"]).all()

	for partial_match in player.matches:
		match = pubg.matches().get(partial_match.id)

loop = asyncio.get_event_loop()
loop.run_until_complete(ent())

```
