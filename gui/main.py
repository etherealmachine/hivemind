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
		self.builder.add_from_file(os.path.join(os.path.dirname(os.path.realpath(__file__)), "gui.glade"))
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
		gobject.idle_add(self.draw)
		
	def set_size(self, size):
		self.size = size
		self.board = [['empty' for i in range(self.size)] for j in range(self.size)]

	def on_back_clicked(self, btn):
		try:
			self.cur.previous()
			self.builder.get_object("forward").set_sensitive(True)
			self.cur.next()
			if 'B' in self.cur.node.data:
				i = self.cur.node['B'][0][0]
				j = self.cur.node['B'][0][1]
			elif 'W' in self.cur.node.data:
				i = self.cur.node['W'][0][0]
				j = self.cur.node['W'][0][1]
			else:
				print 'error'
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
			i, j = ord(i)-97, ord(j)-97
			self.last = (i, j)
		except sgflib.GameTreeEndError:
			self.builder.get_object("back").set_sensitive(False)

	def on_forward_clicked(self, btn):
		try:
			self.cur.next()
			self.builder.get_object("back").set_sensitive(True)
			if 'B' in self.cur.node.data:
				color = 'black'
				i = self.cur.node['B'][0][0]
				j = self.cur.node['B'][0][1]
			elif 'W' in self.cur.node.data:
				color = 'white'
				i = self.cur.node['W'][0][0]
				j = self.cur.node['W'][0][1]
			i, j = ord(i)-97, ord(j)-97
			if color != 'empty':
				if i < self.size and j < self.size:
					self.last = (i, j)
					self.board[i][j] = color
		except sgflib.GameTreeEndError:
			self.builder.get_object("forward").set_sensitive(False)
		
	def on_file_set(self, chooser):
		self.tree = sgflib.SGFParser(open(chooser.get_filename()).read()).parse()[0]
		self.set_size(int(self.tree[0]['SZ'][0]))
		self.cur = self.tree.cursor()
		self.builder.get_object("forward").set_sensitive(True)
		
	def on_window_destroy(self, widget):
		gtk.main_quit()

	def draw(self):
		gobject.idle_add(self.draw)
		w, h = self.screen.get_size()
		self.screen.fill((200, 200, 200))
		
		if not self.size:
			return
		
		black = (0, 0, 0)
		white = (255, 255, 255)
		gray = (200, 200, 200)

		C = 12
		A = 0.5*C
		B = math.sin(1.04719755)*C
		width = 2*B
		height = 2*C
		w, h = self.screen.get_size()
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
		pygame.draw.lines(self.screen, black, False, zip(map(lambda x: x+x_margin, tx), map(lambda y: y+y_margin, ty)), 6)
		pygame.draw.lines(self.screen, black, False, zip(map(lambda x: x+x_margin, bx), map(lambda y: y+y_margin, by)), 6)
		pygame.draw.lines(self.screen, white, False, zip(map(lambda x: x+x_margin, lx), map(lambda y: y+y_margin, ly)), 6)
		pygame.draw.lines(self.screen, white, False, zip(map(lambda x: x+x_margin, rx), map(lambda y: y+y_margin, ry)), 6)
		
		for i in range(self.size):
			for j in range(self.size):
				color = None
				if self.board[i][j] == 'black':
					color = black
				elif self.board[i][j] == 'white':
					color = white
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
						pygame.gfxdraw.filled_circle(self.screen, mx, my, 3, (255, 50, 50))
						pygame.gfxdraw.aacircle(self.screen, mx, my, 3, (255, 50, 50))

		pygame.display.flip()

if __name__ == '__main__':
	gui = GUI()
	gtk.main()
