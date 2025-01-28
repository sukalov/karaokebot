# telegram bots for karaoke
## connected to [karaoke songbook](https://github.com/sukalov/karaoke)

this is a go server, that runs two telegram bots, user and admin. the server accomplishes 2 main tasks:
### managing the line
users choose songs from the songbook, and add themselves to the line. server stores the line in memory and backups in redis. admin can see the line at any moment with `/line` command, and also clear the line with `/clear_line` cmd

### CMS for the songbook
admins can interract with the songbook database right inside the admin bot. after changes are made, `/rebuild` command hits github pages webhook and the songbook rebuilds with updated data
