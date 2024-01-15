Pod::Spec.new do |s|    
    s.name             = 'Libcore'
    s.version          = '0.10.0'
    s.summary          = 'hiddify mobile SDK for iOS'  
    s.homepage         = 'https://hiddify.com/'
    s.license          = { :type => 'Copyright', :text => 'Hiddify Open Software' }
    s.author           = { 'Hiddify' => 'ios@hiddify.com' }
    s.source = { :git => 'https://github.com/hiddify/hiddify-next-core.git', :tag => s.version }
    s.ios.deployment_target = '9.0'
end
  
