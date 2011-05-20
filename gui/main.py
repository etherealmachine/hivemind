#!/usr/bin/env python
import sys
import pygtk  
pygtk.require("2.0")  
import gtk  
import gtk.glade  
import gobject
import pygame
import pygame.gfxdraw
import os
import sgflib
import math

class GUI(object):
	def __init__(self):
		gtk.settings_get_default().props.gtk_button_images = True
		
		self.builder = gtk.Builder()
		self.builder.add_from_file(get_resource_path("gui.glade"))
		self.builder.connect_signals(self)
		
		self.builder.get_object("sgf_file_filter").add_pattern("*.sgf")
		
		self.builder.get_object("forward").set_sensitive(False)
		self.builder.get_object("back").set_sensitive(False)
		
		canvas = self.builder.get_object("canvas")
		os.putenv('SDL_WINDOWID', str(canvas.window.xid))
		gtk.gdk.flush()
		pygame.init()
		pygame.display.set_mode((400, 400), 0, 0)
		self.screen = pygame.display.get_surface()
		self.size = None
		self.last = None
		self.font = pygame.font.Font(get_resource_path("DroidSansMono.ttf"), 16)
		gobject.idle_add(self.draw)
		if len(sys.argv) == 2:
			self.set_file(sys.argv[1])
		
	def set_size(self, size):
		self.size = size
		self.board = [['empty' for i in range(self.size)] for j in range(self.size)]

	def on_back_clicked(self, btn):
		try:
			self.cur.previous()
			self.builder.get_object("forward").set_sensitive(True)
			self.cur.next()
			i, j = -1, -1
			if 'B' in self.cur.node.data:
				if self.cur.node['B'][0] != '':
					i = self.cur.node['B'][0][0]
					j = self.cur.node['B'][0][1]
			elif 'W' in self.cur.node.data:
				if self.cur.node['W'][0] != '':
					i = self.cur.node['W'][0][0]
					j = self.cur.node['W'][0][1]
			else:
				print 'error'
			if i != -1 and j != -1:
				i, j = ord(i)-97, ord(j)-97
			if i < self.size and j < self.size:
				self.board[i][j] = 'empty'
			self.cur.previous()
			if 'B' in self.cur.node.data:
				i = self.cur.node['B'][0][0]
				j = self.cur.node['B'][0][1]
			elif 'W' in self.cur.node.data:
				i = self.cur.node['W'][0][0]
				j = self.cur.node['W'][0][1]
			else:
				self.last = None
				return
			if i != -1 and j != -1:
				i, j = ord(i)-97, ord(j)-97
				j = self.size - j - 1
			self.last = (i, j)
		except sgflib.GameTreeEndError:
			self.builder.get_object("back").set_sensitive(False)

	def on_forward_clicked(self, btn):
		try:
			self.cur.next()
			self.builder.get_object("back").set_sensitive(True)
			if 'B' in self.cur.node.data:
				color = 'black'
				if self.cur.node['B'][0] == '':
					raise sgflib.GameTreeEndError
				i = self.cur.node['B'][0][0]
				j = self.cur.node['B'][0][1]
			elif 'W' in self.cur.node.data:
				color = 'white'
				if self.cur.node['W'][0] == '':
					raise sgflib.GameTreeEndError
				i = self.cur.node['W'][0][0]
				j = self.cur.node['W'][0][1]
			i, j = ord(i)-97, ord(j)-97
			j = self.size - j - 1
			if color != 'empty':
				if i < self.size and j < self.size:
					self.last = (i, j)
					self.board[i][j] = color
		except sgflib.GameTreeEndError:
			self.builder.get_object("forward").set_sensitive(False)
		
	def on_file_set(self, chooser):
		self.set_file(chooser.get_filename())
		
	def on_window_destroy(self, widget):
		gtk.main_quit()
		
	def set_file(self, filename):
		self.tree = sgflib.SGFParser(open(filename).read()).parse()[0]
		self.set_size(int(self.tree[0]['SZ'][0]))
		self.pb = self.tree[0]['PB'][0]
		self.pw = self.tree[0]['PW'][0]
		self.cur = self.tree.cursor()
		self.builder.get_object("forward").set_sensitive(True)

	def draw(self):
		gobject.idle_add(self.draw)
		w, h = self.screen.get_size()
		self.screen.fill((200, 200, 200))
		
		if not self.size:
			return
		
		black = (0, 0, 0)
		white = (255, 255, 255)
		gray = (200, 200, 200)
		red = (255, 0, 0)
		blue = (67, 83, 255)

		C = 10
		A = 0.5*C
		B = math.sin(1.04719755)*C
		width = 2*B
		height = 2*C
		tot_width = self.size * width + self.size*width/2.0
		tot_height = self.size * (A+C)
		x_margin = (w - tot_width) / 2.0
		y_margin = (h - tot_height) / 2.0

		x = [0, 0, B, 2*B, 2*B, B]
		y = [A+C, A, 0, A, A+C, 2*C]
		
		tx, ty, bx, by, lx, ly, rx, ry = [], [], [], [], [], [], [], []
		for i in range(self.size):
			x_off = i * width
			tx += map(lambda x: x+x_off, [0, B, 2*B])
			ty += [A, 0, A]
			bx += map(lambda x: x+x_off+(self.size-1)*(width/2.0), [0, B, 2*B])
			by += map(lambda y: y+tot_height-height+6, [A+C, 2*C, A+C])
			lx += map(lambda x: x+i*(width/2.0), [0, 0, B])
			ly += map(lambda y: y+i*(2*C-A), [A, A+C, 2*C])
			rx += map(lambda x: x+(self.size-1)*width+i*(width/2.0), [B, 2*B, 2*B])
			ry += map(lambda y: y+i*(A+C), [0, A, A+C])
		lx.pop(len(lx)-1)
		ly.pop(len(ly)-1)
		rx.pop(0)
		ry.pop(0)
		pygame.draw.lines(self.screen, red, False, zip(map(lambda x: x+x_margin, tx), map(lambda y: y+y_margin, ty)), 6)
		pygame.draw.lines(self.screen, red, False, zip(map(lambda x: x+x_margin, bx), map(lambda y: y+y_margin, by)), 6)
		pygame.draw.lines(self.screen, blue, False, zip(map(lambda x: x+x_margin, lx), map(lambda y: y+y_margin, ly)), 6)
		pygame.draw.lines(self.screen, blue, False, zip(map(lambda x: x+x_margin, rx), map(lambda y: y+y_margin, ry)), 6)
		for i in range(self.size):
			if chr(97+i) >= 'i':
				c = chr(97+i+1)
			else:
				c = chr(97+i)
			col = self.font.render(c, True, black)
			self.screen.blit(col, (x_margin + i*width, y_margin-col.get_height()))
			row = self.font.render(str(i+1), True, black)
			self.screen.blit(row, (x_margin-row.get_width()-5+i*(width/2.0), y_margin+i*(2*C-A)+5))
		
		for i in range(self.size):
			for j in range(self.size):
				color = None
				if self.board[i][j] == 'black':
					color = red
				elif self.board[i][j] == 'white':
					color = blue
				xoff = x_margin + i * width + j*width/2.0
				yoff = y_margin + j * (A+C)
				pygame.gfxdraw.filled_polygon(self.screen, zip(map(lambda x: x+xoff, x), map(lambda y: y+yoff, y)), gray)
				pygame.gfxdraw.aapolygon(self.screen, zip(map(lambda x: x+xoff, x), map(lambda y: y+yoff, y)), black)
				if color:
					mx, my = int(width/2.0+xoff), int(height/2.0+yoff)
					pygame.gfxdraw.filled_circle(self.screen, mx, my, int(C*0.7), color)
					pygame.gfxdraw.aacircle(self.screen, mx, my, int(C*0.7), color)
				if self.last:
					if i == self.last[0] and j == self.last[1]:
						mx, my = int(width/2.0+xoff), int(height/2.0+yoff)
						pygame.gfxdraw.filled_circle(self.screen, mx, my, 3, black)
						pygame.gfxdraw.aacircle(self.screen, mx, my, 3, black)

		pb = self.font.render('Red: {0}'.format(self.pb), True, red)
		self.screen.blit(pb, (4, 0))
		pw = self.font.render('Blue: {0}'.format(self.pw), True, blue)
		self.screen.blit(pw, (4, 20))

		pygame.display.flip()
		
def get_resource_path(filename):
	return os.path.join(os.path.dirname(os.path.realpath(__file__)), filename)

if __name__ == '__main__':
	gui = GUI()
	gtk.main()
