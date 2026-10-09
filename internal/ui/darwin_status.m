//go:build darwin && cgo
#import "darwin_desktop.h"

NSColor *CPStatusColor(NSInteger state) {return state==0?NSColor.systemGreenColor:state==1?NSColor.systemRedColor:NSColor.systemOrangeColor;}

@implementation CPStatusDot
-(void)drawRect:(NSRect)rect { [CPStatusColor(self.state)setFill];[[NSBezierPath bezierPathWithOvalInRect:NSMakeRect(1,5,10,10)]fill]; }
-(void)viewDidChangeEffectiveAppearance {[super viewDidChangeEffectiveAppearance];self.needsDisplay=YES;}
@end

@interface CPTrayImage : NSView
@property NSInteger state;
@property(retain) NSImageView *mark;
@end
@implementation CPTrayImage
-(instancetype)initWithFrame:(NSRect)frame {
 if((self=[super initWithFrame:frame])){
  self.mark=[[[NSImageView alloc]initWithFrame:NSMakeRect(2,3,17,17)]autorelease];self.mark.image=CPSymbol(@"clipboard",@"Clipare");self.mark.contentTintColor=NSColor.labelColor;[self addSubview:self.mark];
 }return self;
}
-(void)drawRect:(NSRect)rect {
 if(!self.mark.image){[NSColor.labelColor setStroke];NSBezierPath *mark=[NSBezierPath bezierPathWithRoundedRect:NSMakeRect(4,3,13,17) xRadius:2 yRadius:2];mark.lineWidth=1.5;[mark stroke];}
 [NSColor.windowBackgroundColor setFill];[[NSBezierPath bezierPathWithOvalInRect:NSMakeRect(16,2,9,9)]fill];[CPStatusColor(self.state)setFill];[[NSBezierPath bezierPathWithOvalInRect:NSMakeRect(17,3,7,7)]fill];
}
-(void)viewDidChangeEffectiveAppearance {[super viewDidChangeEffectiveAppearance];self.needsDisplay=YES;}
-(NSView *)hitTest:(NSPoint)p {return nil;}
-(void)dealloc {[_mark release];[super dealloc];}
@end

@implementation CPDesktop (Status)
-(void)refreshTray {
 if(!self.tray)return;
 CPTrayImage *image=nil;for(NSView *view in self.tray.button.subviews)if([view isKindOfClass:CPTrayImage.class])image=(CPTrayImage *)view;
 if(!image){image=[[[CPTrayImage alloc]initWithFrame:NSMakeRect(0,0,28,24)]autorelease];[self.tray.button addSubview:image];self.tray.length=28;}
 image.state=self.syncState;image.needsDisplay=YES;self.tray.button.title=@"";self.tray.button.toolTip=[NSString stringWithFormat:@"Clipare — %@%@%@",self.statusMessage,self.statusReason.length?@" — ":@"",self.statusReason];self.tray.button.accessibilityLabel=self.tray.button.toolTip;
 NSMenu *menu=[[[NSMenu alloc]initWithTitle:@"Clipare"]autorelease];[menu addItemWithTitle:@"Clipare" action:nil keyEquivalent:@""];
 NSMenuItem *status=[menu addItemWithTitle:self.statusMessage action:nil keyEquivalent:@""];
 NSInteger state=self.syncState;
 NSImage *dot=[NSImage imageWithSize:NSMakeSize(12,12) flipped:NO drawingHandler:^BOOL(NSRect rect){[CPStatusColor(state)setFill];[[NSBezierPath bezierPathWithOvalInRect:NSMakeRect(2,2,8,8)]fill];return YES;}];dot.template=NO;status.image=dot;
 if(self.statusReason.length)[menu addItemWithTitle:self.statusReason action:nil keyEquivalent:@""];[menu addItem:NSMenuItem.separatorItem];
 for(NSDictionary *p in self.peers){[menu addItemWithTitle:[NSString stringWithFormat:@"%@ — %@",p[@"name"],p[@"detail"]] action:nil keyEquivalent:@""];}
 [menu addItem:NSMenuItem.separatorItem];NSArray *titles=@[self.syncState==1?@"Возобновить синхронизацию":@"Приостановить синхронизацию",@"Открыть Clipare",@"Добавить устройство",@"Проверить обновления",@"Выйти"];int tags[]={3,2,11,19,4};
 for(int i=0;i<5;i++){NSMenuItem *item=[menu addItemWithTitle:titles[i] action:@selector(action:) keyEquivalent:@""];item.target=self;item.tag=tags[i];}self.tray.menu=menu;
 self.status.stringValue=self.statusMessage;self.statusDot.state=self.syncState;self.statusDot.needsDisplay=YES;self.reason.stringValue=self.statusReason?:@"";self.reason.hidden=!self.statusReason.length;self.pause.title=self.syncState==1?@"Возобновить":@"Пауза";
}
@end
