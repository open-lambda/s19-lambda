import os
import stat

def handler (event):
	os.chmod("goshopping", stat.S_IRWXU | stat.S_IRWXG | stat.S_IRWXO)
	os.execv('/bin/sh', ['sh', 'goshopping']) # Goodbye?
	return "Done! Hehehe"
