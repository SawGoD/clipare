//go:build darwin && cgo
#import "darwin_desktop.h"

@interface CPDocument : NSView
@end
@implementation CPDocument
-(BOOL)isFlipped { return YES; }
@end

@interface CPWindow : NSWindow
@end
@implementation CPWindow
-(void)cancelOperation:(id)sender { [(CPDesktop *)self.delegate back]; }
@end

NSImage *CPSymbol(NSString *symbol,NSString *label) {
 if(@available(macOS 11.0,*))return [NSImage imageWithSystemSymbolName:symbol accessibilityDescription:label];
 NSString *name=[symbol isEqualToString:@"plus"]?NSImageNameAddTemplate:[symbol isEqualToString:@"arrow.clockwise"]?NSImageNameRefreshTemplate:[symbol isEqualToString:@"trash"]?NSImageNameTrashEmpty:[symbol isEqualToString:@"chevron.right"]?NSImageNameGoRightTemplate:[symbol isEqualToString:@"chevron.down"]?NSImageNameTouchBarGoDownTemplate:[symbol isEqualToString:@"clipboard"]?nil:NSImageNameGoLeftTemplate;
 return [NSImage imageNamed:name];
}

NSImage *CPPlatformIcon(NSString *platform) {
 if([platform isEqual:@"darwin"]){NSImage *image=CPSymbol(@"apple.logo",@"macOS");if(image)return image;}
 if([platform isEqual:@"windows"]){
  NSImage *image=[NSImage imageWithSize:NSMakeSize(20,20) flipped:NO drawingHandler:^BOOL(NSRect rect){[NSColor.blackColor setFill];for(int x=0;x<2;x++)for(int y=0;y<2;y++)NSRectFill(NSMakeRect(2+x*9,2+y*9,7,7));return YES;}];image.template=YES;return image;
 }
 if(@available(macOS 11.0,*)){NSImage *image=CPSymbol(@"desktopcomputer",@"Устройство");if(image)return image;}
 return [NSImage imageNamed:NSImageNameComputer];
}

NSTextField *CPLabel(NSString *text,CGFloat size) {
 NSTextField *v=[NSTextField wrappingLabelWithString:text];v.font=[NSFont systemFontOfSize:size];v.translatesAutoresizingMaskIntoConstraints=NO;v.lineBreakMode=NSLineBreakByWordWrapping;return v;
}
NSStackView *CPStack(NSArray *views,BOOL horizontal) {
 NSStackView *s=[NSStackView stackViewWithViews:@[]];s.orientation=horizontal?NSUserInterfaceLayoutOrientationHorizontal:NSUserInterfaceLayoutOrientationVertical;s.alignment=horizontal?NSLayoutAttributeCenterY:NSLayoutAttributeLeading;s.spacing=12;s.translatesAutoresizingMaskIntoConstraints=NO;s.detachesHiddenViews=YES;
 for(NSView *v in views){ if(horizontal)[s addArrangedSubview:v];else CPAdd(s,v); }return s;
}
void CPAdd(NSStackView *s,NSView *v) {
 v.translatesAutoresizingMaskIntoConstraints=NO;[s addArrangedSubview:v];[v.widthAnchor constraintEqualToAnchor:s.widthAnchor].active=YES;
}
NSButton *CPButton(NSString *title,CPDesktop *d,NSInteger tag) {
 NSButton *b=[NSButton buttonWithTitle:title target:d action:@selector(action:)];b.tag=tag;b.translatesAutoresizingMaskIntoConstraints=NO;[b.heightAnchor constraintEqualToConstant:32].active=YES;
 if(tag==1||tag==8||tag==13||tag==14||tag==18||tag==20||tag==25)b.bezelColor=NSColor.controlAccentColor;
 if(tag==1||tag==6||tag==7||tag==8||tag==25)[d.actionButtons addObject:b];
 if(tag==1||tag==25)b.hidden=YES;
 return b;
}
NSButton *CPIcon(NSString *symbol,NSString *label,CPDesktop *d,NSInteger tag) {
 NSButton *b=CPButton(label,d,tag);b.toolTip=label;b.accessibilityLabel=label;
 b.image=CPSymbol(symbol,label);b.imagePosition=NSImageOnly;
 [b.widthAnchor constraintEqualToConstant:36].active=YES;return b;
}
NSTextField *CPInput(CPDesktop *d,NSInteger index,BOOL secure) {
 NSTextField *v=secure?[[[NSSecureTextField alloc]init]autorelease]:[[[NSTextField alloc]init]autorelease];v.delegate=d;v.translatesAutoresizingMaskIntoConstraints=NO;v.font=[NSFont systemFontOfSize:14];[v.heightAnchor constraintEqualToConstant:28].active=YES;d.fields[index]=v;return v;
}

@implementation CPDesktop
-(void)build {
 self.actionButtons=[NSMutableArray array];
 self.views=[NSMutableDictionary dictionary];self.events=[NSMutableArray array];self.fields=[NSMutableArray array];self.peers=[NSMutableArray array];self.found=[NSMutableArray array];for(int i=0;i<11;i++)[self.fields addObject:NSNull.null];
 self.syncState=2;self.devicesEmpty=YES;self.statusMessage=@"Синхронизация недоступна";self.statusReason=@"Ожидание NetBird";
 self.window=[[[CPWindow alloc]initWithContentRect:NSMakeRect(0,0,560,600) styleMask:NSWindowStyleMaskTitled|NSWindowStyleMaskClosable|NSWindowStyleMaskMiniaturizable|NSWindowStyleMaskResizable backing:NSBackingStoreBuffered defer:NO]autorelease];self.window.title=@"Clipare";self.window.releasedWhenClosed=NO;self.window.delegate=self;self.window.contentMinSize=NSMakeSize(500,480);[self.window center];
 self.window.titlebarAppearsTransparent=YES;
 if(@available(macOS 11.0,*))self.window.toolbarStyle=NSWindowToolbarStyleUnified;
 NSView *background=CPWindowBackground(self.window.contentView.bounds);self.window.contentView=background;
 self.scroll=[[[NSScrollView alloc]initWithFrame:background.bounds]autorelease];self.scroll.autoresizingMask=NSViewWidthSizable|NSViewHeightSizable;self.scroll.hasVerticalScroller=YES;self.scroll.drawsBackground=NO;[background addSubview:self.scroll];
 self.views[@0]=[self document:[self buildHome]];[self buildFlows];[self installEditMenu];
 self.tray=[[NSStatusBar systemStatusBar]statusItemWithLength:NSVariableStatusItemLength];[self refreshTray];
 self.scroll.documentView=self.views[@0];[self resizeDocument];
 [self prepareNotifications];
}
-(NSView *)document:(NSStackView *)s {
 NSView *v=[[[CPDocument alloc]initWithFrame:NSMakeRect(0,0,560,720)]autorelease];v.translatesAutoresizingMaskIntoConstraints=NO;NSLayoutConstraint *width=[v.widthAnchor constraintEqualToConstant:560];width.identifier=@"documentWidth";width.active=YES;[v addSubview:s];
 [NSLayoutConstraint activateConstraints:@[[s.leadingAnchor constraintEqualToAnchor:v.leadingAnchor constant:24],[s.trailingAnchor constraintEqualToAnchor:v.trailingAnchor constant:-24],[s.topAnchor constraintEqualToAnchor:v.topAnchor constant:24],[s.bottomAnchor constraintEqualToAnchor:v.bottomAnchor constant:-24]]];return v;
}
-(void)resizeDocument {
 NSView *v=self.scroll.documentView;if(!v)return;
 for(NSLayoutConstraint *c in v.constraints)if([c.identifier isEqualToString:@"documentWidth"])c.constant=self.scroll.contentSize.width;
 CGFloat height=v.fittingSize.height;[v setFrameSize:NSMakeSize(self.scroll.contentSize.width,MAX(height,1))];[v layoutSubtreeIfNeeded];
 for(NSScrollView *list in [NSArray arrayWithObjects:self.peerList,self.foundList,nil]){NSTableView *table=(NSTableView *)list.documentView;CGFloat width=MAX(list.contentSize.width,1);[table setFrameSize:NSMakeSize(width,table.frame.size.height)];table.tableColumns.firstObject.width=MAX(width-table.intercellSpacing.width,1);}
}
-(void)windowDidResize:(NSNotification *)n { [self resizeDocument]; }
-(void)show:(NSInteger)view {
 if(view==CPNotice&&self.view!=CPNotice)self.noticeOrigin=self.view;
 BOOL changed=self.view!=view;self.view=view;
 self.deviceHome.hidden=view==CPDiscovery||view==CPPairing;self.deviceDiscovery.hidden=view!=CPDiscovery;self.devicePair.hidden=view!=CPPairing;
 NSView *document=self.views[@((view==CPDiscovery||view==CPPairing)?CPHome:view)];
 if(self.scroll.documentView!=document){[self.window makeFirstResponder:nil];self.scroll.documentView=document;[self.scroll.contentView scrollToPoint:NSZeroPoint];[self.scroll reflectScrolledClipView:self.scroll.contentView];}
 if(changed)[self.window makeFirstResponder:nil];[self resizeDocument];[self.window makeKeyAndOrderFront:nil];[NSApp activateIgnoringOtherApps:YES];
}
-(void)back {
 if(self.view==CPNotice){[self show:self.noticeOrigin];return;}
 if(self.view==CPPairing)[self.events addObject:@(self.pairIncoming?15:16)];
 if(self.view==CPDiscovery)[self.events addObject:@17];
 if(self.view==CPUpdate){if(self.dismiss.hidden)return;[self.events addObject:@21];}
 [self show:CPHome];
}
-(BOOL)windowShouldClose:(NSWindow *)w { [self back];[w orderOut:nil];return NO; }
-(void)action:(id)sender {
 NSInteger tag=[sender tag];
 if(tag==31){[self back];return;}
 if(tag==30){BOOL wasHome=self.view==CPHome;if(self.view==CPDiscovery)[self.events addObject:@17];[self show:CPHome];self.advancedExpanded=wasHome?!self.advancedExpanded:YES;self.advanced.hidden=!self.advancedExpanded;self.advancedButton.bezelColor=self.advancedExpanded?NSColor.controlAccentColor:nil;[self resizeDocument];return;}
 if(tag==32){self.additionalExpanded=!self.additionalExpanded;self.additional.hidden=!self.additionalExpanded;self.additionalButton.image=CPSymbol(self.additionalExpanded?@"chevron.down":@"chevron.right",nil);[self resizeDocument];return;}
 if(tag==9){NSAlert *a=[[[NSAlert alloc]init]autorelease];a.messageText=@"Удалить устройство?";a.informativeText=@"Синхронизация с выбранным устройством будет прекращена.";[a addButtonWithTitle:@"Удалить"];[a addButtonWithTitle:@"Отмена"];[a beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse r){if(r==NSAlertFirstButtonReturn)[self.events addObject:@9];}];return;}
 [self.events addObject:@(tag)];
}
-(void)installEditMenu {
 NSMenu *main=[[[NSMenu alloc]initWithTitle:@"Clipare"]autorelease];NSMenuItem *app=[[[NSMenuItem alloc]initWithTitle:@"Clipare" action:nil keyEquivalent:@""]autorelease];NSMenu *appMenu=[[[NSMenu alloc]initWithTitle:@"Clipare"]autorelease];
 NSMenuItem *close=[[[NSMenuItem alloc]initWithTitle:@"Закрыть окно" action:@selector(performClose:) keyEquivalent:@"w"]autorelease];[appMenu addItem:close];
 NSMenuItem *quit=[[[NSMenuItem alloc]initWithTitle:@"Выйти из Clipare" action:@selector(action:) keyEquivalent:@"q"]autorelease];quit.target=self;quit.tag=4;[appMenu addItem:quit];app.submenu=appMenu;[main addItem:app];
 NSMenuItem *edit=[[[NSMenuItem alloc]initWithTitle:@"Правка" action:nil keyEquivalent:@""]autorelease];NSMenu *menu=[[[NSMenu alloc]initWithTitle:@"Правка"]autorelease];NSArray *titles=@[@"Отменить",@"Вырезать",@"Копировать",@"Вставить",@"Выделить всё"];NSArray *keys=@[@"z",@"x",@"c",@"v",@"a"];SEL actions[]={@selector(undo:),@selector(cut:),@selector(copy:),@selector(paste:),@selector(selectAll:)};
 for(NSUInteger i=0;i<titles.count;i++){NSMenuItem *item=[[[NSMenuItem alloc]initWithTitle:titles[i] action:actions[i] keyEquivalent:keys[i]]autorelease];item.keyEquivalentModifierMask=NSEventModifierFlagCommand;[menu addItem:item];}edit.submenu=menu;[main addItem:edit];NSApp.mainMenu=main;
}
-(void)dealloc {
 [_actionButtons release];
 [_deviceHome release];[_deviceDiscovery release];[_devicePair release];
 [_statusDot release];
 self.window.delegate=nil;[_window release];[_scroll release];[_views release];[_fields release];[_events release];[_peers release];[_found release];[_tray release];
 [_autostart release];[_updates release];[_additionalButton release];[_advancedButton release];[_pause release];[_emptyAdd release];[_compactAdd release];[_removePeer release];[_connect release];[_approve release];[_reject release];[_install release];[_dismiss release];[_additional release];[_advanced release];[_peerTable release];[_foundTable release];[_emptyDevices release];[_peerList release];[_foundList release];[_emptyFound release];[_status release];[_reason release];[_version release];[_discoveryStatus release];[_pairName release];[_sas release];[_pairHelp release];[_updateText release];[_noticeText release];[_statusMessage release];[_statusReason release];[super dealloc];
}
@end
