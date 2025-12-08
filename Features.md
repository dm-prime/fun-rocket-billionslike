plan a space shooter game with ebiten that can scale to a lot of enemies really easily

- preallocate a bunch of cells where enemies can eixst in, enough for a gigantic map
- playeres, enemies and projectiles always exist in a cell so that our collision checking will scale ok
-  player is the same entity as an enemy with some interfaces that replace player input with 
- huge radar in the middle of the screen for the player that shows enemies off-screen, but its overlayed on top of the player gfx
- placeholder images for enemy and player and projectiles prrendered from code 