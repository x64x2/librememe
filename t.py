import pubg_python
import asyncio
from rich import print

async def ent():

	p = pubg_python.AioPUBG("", shard=pubg_python.Shard.STEAM)

	ps = p.players().filter(player_names=["auteuil"])
	print(type(ps))
	def participant_in_roster(with_roster, and_id):
		filtered = filter(lambda participant: participant.player_id == and_id, with_roster.participants)
		return len(list(filtered)) != 0
	# ps = await ps.gg()
	pp = (await ps[0])

	print(await pp.matches.all())
	
	for i in pp.matches:
		mm = (await p.matches().get(i.id))
		roster = list(filter(lambda roaster: participant_in_roster(roaster, pp.id), mm.rosters))[0]
		for rr in roster.participants:
			print(rr.kills)
l = asyncio.get_event_loop()
l.run_until_complete(ent())
