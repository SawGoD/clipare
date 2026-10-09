//go:build darwin && cgo
#import "darwin_desktop.h"

// CGColor is a snapshot, unlike NSColor. Resolve it in this view's appearance,
// not the process-wide appearance (which can differ from the window's).
static void CPInAppearance(NSView *view,void (^draw)(void)) {
 if(@available(macOS 11.0,*)){[view.effectiveAppearance performAsCurrentDrawingAppearance:draw];return;}
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
 NSAppearance *previous=NSAppearance.currentAppearance;NSAppearance.currentAppearance=view.effectiveAppearance;draw();NSAppearance.currentAppearance=previous;
#pragma clang diagnostic pop
}

// One native material implementation for cards. No custom blur or fixed theme.
@interface CPGlassSurface : NSView
@property(retain) NSView *body;
@property(retain) NSView *effect;
@property(retain) NSArray *bodyConstraints;
@property BOOL forceFallback;
-(void)refreshMaterial;
-(void)applyReducedTransparency:(BOOL)opaque;
@end

@implementation CPGlassSurface
-(instancetype)initWithBody:(NSView *)body {
 if((self=[super initWithFrame:NSZeroRect])){
  self.translatesAutoresizingMaskIntoConstraints=NO;self.body=body;
  [NSWorkspace.sharedWorkspace.notificationCenter addObserver:self selector:@selector(refreshMaterial) name:NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification object:nil];
  [self refreshMaterial];
 }return self;
}
-(void)refreshMaterial {
 [self applyReducedTransparency:NSWorkspace.sharedWorkspace.accessibilityDisplayShouldReduceTransparency];
}
-(void)applyReducedTransparency:(BOOL)opaque {
 if(self.bodyConstraints)[NSLayoutConstraint deactivateConstraints:self.bodyConstraints];
 [self.body removeFromSuperview];[self.effect removeFromSuperview];self.effect=nil;
 NSView *host=self;
 BOOL glassHost=NO;
 if(!opaque){
#if __MAC_OS_X_VERSION_MAX_ALLOWED >= 260000
  if(@available(macOS 26.0,*)){if(!self.forceFallback){
   NSGlassEffectView *glass=[[[NSGlassEffectView alloc]init]autorelease];glass.cornerRadius=20;glass.style=NSGlassEffectViewStyleRegular;glass.contentView=self.body;self.effect=glass;glassHost=YES;
  }}
#endif
  if(!self.effect){
   NSVisualEffectView *material=[[[NSVisualEffectView alloc]init]autorelease];material.material=NSVisualEffectMaterialPopover;material.blendingMode=NSVisualEffectBlendingModeWithinWindow;material.state=NSVisualEffectStateFollowsWindowActiveState;
   material.wantsLayer=YES;material.layer.cornerRadius=20;material.layer.masksToBounds=YES;self.effect=material;
  }
  self.effect.translatesAutoresizingMaskIntoConstraints=NO;[self addSubview:self.effect];
  [NSLayoutConstraint activateConstraints:@[[self.effect.leadingAnchor constraintEqualToAnchor:self.leadingAnchor],[self.effect.trailingAnchor constraintEqualToAnchor:self.trailingAnchor],[self.effect.topAnchor constraintEqualToAnchor:self.topAnchor],[self.effect.bottomAnchor constraintEqualToAnchor:self.bottomAnchor]]];
  host=self.effect;
 }
 // Glass owns the placement of contentView in its internal hierarchy.
 if(!glassHost)[host addSubview:self.body];
 self.body.translatesAutoresizingMaskIntoConstraints=NO;
 self.bodyConstraints=@[[self.body.leadingAnchor constraintEqualToAnchor:host.leadingAnchor],[self.body.trailingAnchor constraintEqualToAnchor:host.trailingAnchor],[self.body.topAnchor constraintEqualToAnchor:host.topAnchor],[self.body.bottomAnchor constraintEqualToAnchor:host.bottomAnchor]];
 [NSLayoutConstraint activateConstraints:self.bodyConstraints];self.wantsLayer=YES;self.layer.cornerRadius=20;
 CPInAppearance(self,^{self.layer.backgroundColor=(opaque?NSColor.controlBackgroundColor:NSColor.clearColor).CGColor;self.layer.borderWidth=opaque?1:0;self.layer.borderColor=NSColor.separatorColor.CGColor;});
}
-(void)viewDidChangeEffectiveAppearance {[super viewDidChangeEffectiveAppearance];[self refreshMaterial];}
-(void)dealloc {
 [NSWorkspace.sharedWorkspace.notificationCenter removeObserver:self];[_body release];[_effect release];[_bodyConstraints release];[super dealloc];
}
@end

NSView *CPCard(NSStackView *stack) {
 NSView *body=[[[NSView alloc]init]autorelease];body.translatesAutoresizingMaskIntoConstraints=NO;[body addSubview:stack];
 [NSLayoutConstraint activateConstraints:@[[stack.leadingAnchor constraintEqualToAnchor:body.leadingAnchor constant:20],[stack.trailingAnchor constraintEqualToAnchor:body.trailingAnchor constant:-20],[stack.topAnchor constraintEqualToAnchor:body.topAnchor constant:18],[stack.bottomAnchor constraintEqualToAnchor:body.bottomAnchor constant:-18]]];
 return [[[CPGlassSurface alloc]initWithBody:body]autorelease];
}

@interface CPWindowMaterial : NSVisualEffectView
@end
@implementation CPWindowMaterial
-(instancetype)initWithFrame:(NSRect)frame {
 if((self=[super initWithFrame:frame])){
  self.material=NSVisualEffectMaterialUnderWindowBackground;self.blendingMode=NSVisualEffectBlendingModeBehindWindow;self.state=NSVisualEffectStateFollowsWindowActiveState;
  [NSWorkspace.sharedWorkspace.notificationCenter addObserver:self selector:@selector(refreshAccessibility) name:NSWorkspaceAccessibilityDisplayOptionsDidChangeNotification object:nil];[self refreshAccessibility];
 }return self;
}
-(void)refreshAccessibility {self.state=NSWorkspace.sharedWorkspace.accessibilityDisplayShouldReduceTransparency?NSVisualEffectStateInactive:NSVisualEffectStateFollowsWindowActiveState;self.wantsLayer=YES;CPInAppearance(self,^{self.layer.backgroundColor=(NSWorkspace.sharedWorkspace.accessibilityDisplayShouldReduceTransparency?NSColor.windowBackgroundColor:NSColor.clearColor).CGColor;});}
-(void)viewDidChangeEffectiveAppearance {[super viewDidChangeEffectiveAppearance];[self refreshAccessibility];}
-(void)dealloc {[NSWorkspace.sharedWorkspace.notificationCenter removeObserver:self];[super dealloc];}
@end

NSView *CPWindowBackground(NSRect frame) {return [[[CPWindowMaterial alloc]initWithFrame:frame]autorelease];}
